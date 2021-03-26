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

package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/cli/mobycli/resolvepath"
	"github.com/docker/compose-cli/utils"
)

var (
	boolflags = []string{
		"--debug", "-D",
		"--verbose",
		"--log-level",
		"--l",
		"--tls",
		"--tlsverivy",
	}

	stringflags = []string{
		"--tlscacert",
		"--tlscert",
		"--tlskey",
		"--host", "-H",
		"--context",
	}
)

func main() {
	root := &cobra.Command{
		DisableFlagParsing: true,
		Use:                "docker-compose",
		Run: func(cmd *cobra.Command, args []string) {
			if _, ok := os.LookupEnv("DOCKER_COMPOSE_USE_V1"); ok {
				runComposeV1(args)
			}

			compose := convert(args)
			runComposeV2(compose)
		},
	}

	err := root.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func convert(args []string) []string {
	rootFlags := []string{}
	command := []string{"compose"}
	l := len(args)
	for i := 0; i < l; i++ {
		arg := args[i]
		if arg == "--verbose" {
			arg = "--debug"
		}
		if arg == "-h" {
			// docker cli has deprecated -h to avoid ambiguity with -H, while docker-compose still support it
			arg = "--help"
		}
		if arg == "--version" {
			// redirect --version pseudo-command to actual command
			arg = "version"
		}
		if utils.StringContains(boolflags, arg) {
			rootFlags = append(rootFlags, arg)
			continue
		}
		if utils.StringContains(stringflags, arg) {
			i++
			if i >= l {
				fmt.Fprintf(os.Stderr, "flag needs an argument: '%s'\n", arg)
				os.Exit(1)
			}
			rootFlags = append(rootFlags, arg, args[i])
			continue
		}
		command = append(command, arg)
	}
	return append(rootFlags, command...)
}

func runComposeV1(args []string) {
	execBinary, err := resolvepath.LookPath("docker-compose-v1")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = syscall.Exec(execBinary, append([]string{"docker-compose"}, args...), os.Environ())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runComposeV2(args []string) {
	execBinary, err := resolvepath.LookPath("docker")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = syscall.Exec(execBinary, append([]string{"docker"}, args...), append(os.Environ(), "DOCKER_METRICS_SOURCE=docker-compose"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
