/*
   Copyright 2020 Docker Compose CLI authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package mobycli

import (
	"context"
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/spf13/cobra"

	apicontext "github.com/docker/compose-cli/api/context"
	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/cli/metrics"
	"github.com/docker/compose-cli/cli/mobycli/resolvepath"
)

var delegatedContextTypes = []string{store.DefaultContextType}

// ComDockerCli name of the classic cli binary
var ComDockerCli = "com.docker.cli"

func init() {
	if runtime.GOOS == "windows" {
		ComDockerCli += ".exe"
	}
}

// ExecIfDefaultCtxType delegates to com.docker.cli if on moby context
func ExecIfDefaultCtxType(ctx context.Context, root *cobra.Command) {
	currentContext := apicontext.Current()

	s := store.Instance()

	currentCtx, err := s.Get(currentContext)
	// Only run original docker command if the current context is not ours.
	if err != nil || mustDelegateToMoby(currentCtx.Type()) {
		Exec(root)
	}
}

func mustDelegateToMoby(ctxType string) bool {
	for _, ctype := range delegatedContextTypes {
		if ctxType == ctype {
			return true
		}
	}
	return false
}

// Exec delegates to com.docker.cli if on moby context
func Exec(_ *cobra.Command) {
	metricsClient := metrics.NewDefaultClient()
	metricsClient.WithCliVersionFunc(func() string {
		return CliVersion()
	})
	start := time.Now().UTC()
	childExit := make(chan bool)
	err := RunDocker(childExit, os.Args[1:]...)
	childExit <- true
	duration := time.Since(start)
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			exitCode := exiterr.ExitCode()
			metricsClient.Track(
				metrics.CmdResult{
					ContextType: store.DefaultContextType,
					Args:        os.Args[1:],
					Status:      metrics.FailureCategoryFromExitCode(exitCode).MetricsStatus,
					ExitCode:    exitCode,
					Start:       start,
					Duration:    duration,
				},
			)
			os.Exit(exitCode)
		}
		metricsClient.Track(
			metrics.CmdResult{
				ContextType: store.DefaultContextType,
				Args:        os.Args[1:],
				Status:      metrics.FailureStatus,
				Start:       start,
				Duration:    duration,
			},
		)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	commandArgs := os.Args[1:]
	command := metrics.GetCommand(commandArgs)
	if command == "login" && !metrics.HasQuietFlag(commandArgs) {
		displayPATSuggestMsg(commandArgs)
	}
	metricsClient.Track(
		metrics.CmdResult{
			ContextType: store.DefaultContextType,
			Args:        os.Args[1:],
			Status:      metrics.SuccessStatus,
			ExitCode:    0,
			Start:       start,
			Duration:    duration,
		},
	)

	os.Exit(0)
}

// RunDocker runs a docker command, and forward signals to the shellout command (stops listening to signals when an event is sent to childExit)
func RunDocker(childExit chan bool, args ...string) error {
	cmd := exec.Command(comDockerCli(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	signals := make(chan os.Signal, 1)
	signal.Notify(signals) // catch all signals
	go func() {
		for {
			select {
			case sig := <-signals:
				if cmd.Process == nil {
					continue // can happen if receiving signal before the process is actually started
				}
				// In go1.14+, the go runtime issues SIGURG as an interrupt to
				// support preemptable system calls on Linux. Since we can't
				// forward that along we'll check that here.
				if isRuntimeSig(sig) {
					continue
				}
				_ = cmd.Process.Signal(sig)
			case <-childExit:
				return
			}
		}
	}()

	return cmd.Run()
}

func comDockerCli() string {
	if v := os.Getenv("DOCKER_COM_DOCKER_CLI"); v != "" {
		return v
	}

	execBinary := findBinary(ComDockerCli)
	if execBinary == "" {
		var err error
		execBinary, err = resolvepath.LookPath(ComDockerCli)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, "Current PATH : "+os.Getenv("PATH"))
			os.Exit(1)
		}
	}

	return execBinary
}

func findBinary(filename string) string {
	currentBinaryPath, err := os.Executable()
	if err != nil {
		return ""
	}
	currentBinaryPath, err = filepath.EvalSymlinks(currentBinaryPath)
	if err != nil {
		return ""
	}
	binaryPath := filepath.Join(filepath.Dir(currentBinaryPath), filename)
	if _, err := os.Stat(binaryPath); err != nil {
		return ""
	}
	return binaryPath
}

// IsDefaultContextCommand checks if the command exists in the classic cli (issues a shellout --help)
func IsDefaultContextCommand(dockerCommand string) bool {
	cmd := exec.Command(comDockerCli(), dockerCommand, "--help")
	b, e := cmd.CombinedOutput()
	if e != nil {
		fmt.Println(e)
	}
	return regexp.MustCompile("Usage:\\s*docker\\s*" + dockerCommand).Match(b)
}

// CliVersion returns the docker cli version
func CliVersion() string {
	info, err := buildinfo.ReadFile(ComDockerCli)
	if err != nil {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key != "-ldflags" {
			continue
		}
		args, err := shlex.Split(s.Value)
		if err != nil {
			return ""
		}
		for _, a := range args {
			// https://github.com/docker/cli/blob/f1615facb1ca44e4336ab20e621315fc2cfb845a/scripts/build/.variables#L77
			if !strings.HasPrefix(a, "github.com/docker/cli/cli/version.Version") {
				continue
			}
			parts := strings.Split(a, "=")
			if len(parts) != 2 {
				return ""
			}
			return parts[1]
		}
	}
	return ""
}

// ExecSilent executes a command and do redirect output to stdOut, return output
func ExecSilent(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 0 {
		args = os.Args[1:]
	}
	cmd := exec.CommandContext(ctx, comDockerCli(), args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
