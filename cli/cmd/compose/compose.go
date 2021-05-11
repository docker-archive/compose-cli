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

package compose

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
	dockercli "github.com/docker/cli/cli"
	"github.com/morikuni/aec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/docker/compose-cli/api/compose"
	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/api/errdefs"
	"github.com/docker/compose-cli/cli/formatter"
	"github.com/docker/compose-cli/cli/metrics"
)

// Command defines a compose CLI command as a func with args
type Command func(context.Context, []string) error

// Adapt a Command func to cobra library
func Adapt(fn Command) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		contextString := fmt.Sprintf("%s", ctx)
		if !strings.HasSuffix(contextString, ".WithCancel") { // need to handle cancel
			cancellableCtx, cancel := context.WithCancel(cmd.Context())
			ctx = cancellableCtx
			s := make(chan os.Signal, 1)
			signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)
			go func() {
				<-s
				cancel()
			}()
		}
		err := fn(ctx, args)
		var composeErr metrics.ComposeError
		if errdefs.IsErrCanceled(err) || errors.Is(ctx.Err(), context.Canceled) {
			err = dockercli.StatusError{
				StatusCode: 130,
				Status:     metrics.CanceledStatus,
			}
		}
		if errors.As(err, &composeErr) {
			err = dockercli.StatusError{
				StatusCode: composeErr.GetMetricsFailureCategory().ExitCode,
				Status:     err.Error(),
			}
		}
		return err
	}
}

// Warning is a global warning to be displayed to user on command failure
var Warning string

type projectOptions struct {
	ProjectName string
	Profiles    []string
	ConfigPaths []string
	WorkDir     string
	ProjectDir  string
	EnvFile     string
}

func (o *projectOptions) addProjectFlags(f *pflag.FlagSet) {
	f.StringArrayVar(&o.Profiles, "profile", []string{}, "Specify a profile to enable")
	f.StringVarP(&o.ProjectName, "project-name", "p", "", "Project name")
	f.StringArrayVarP(&o.ConfigPaths, "file", "f", []string{}, "Compose configuration files")
	f.StringVar(&o.EnvFile, "env-file", "", "Specify an alternate environment file.")
	f.StringVar(&o.ProjectDir, "project-directory", "", "Specify an alternate working directory\n(default: the path of the Compose file)")
	f.StringVar(&o.WorkDir, "workdir", "", "DEPRECATED! USE --project-directory INSTEAD.\nSpecify an alternate working directory\n(default: the path of the Compose file)")
	_ = f.MarkHidden("workdir")
}

func (o *projectOptions) toProjectName() (string, error) {
	if o.ProjectName != "" {
		return o.ProjectName, nil
	}

	project, err := o.toProject(nil)
	if err != nil {
		return "", err
	}
	return project.Name, nil
}

func (o *projectOptions) toProject(services []string, po ...cli.ProjectOptionsFn) (*types.Project, error) {
	options, err := o.toProjectOptions(po...)
	if err != nil {
		return nil, metrics.WrapComposeError(err)
	}

	project, err := cli.ProjectFromOptions(options)
	if err != nil {
		return nil, metrics.WrapComposeError(err)
	}

	if len(services) > 0 {
		s, err := project.GetServices(services...)
		if err != nil {
			return nil, err
		}
		o.Profiles = append(o.Profiles, s.GetProfiles()...)
	}

	if profiles, ok := options.Environment["COMPOSE_PROFILES"]; ok {
		o.Profiles = append(o.Profiles, strings.Split(profiles, ",")...)
	}

	project.ApplyProfiles(o.Profiles)

	err = project.ForServices(services)
	return project, err
}

func (o *projectOptions) toProjectOptions(po ...cli.ProjectOptionsFn) (*cli.ProjectOptions, error) {
	return cli.NewProjectOptions(o.ConfigPaths,
		append(po,
			cli.WithEnvFile(o.EnvFile),
			cli.WithDotEnv,
			cli.WithOsEnv,
			cli.WithWorkingDirectory(o.ProjectDir),
			cli.WithDefaultConfigPath,
			cli.WithName(o.ProjectName))...)
}

// RootCommand returns the compose command with its child commands
func RootCommand(contextType string, backend compose.Service) *cobra.Command {
	opts := projectOptions{}
	var ansi string
	var noAnsi bool
	command := &cobra.Command{
		Short:            "Docker Compose",
		Use:              "compose",
		TraverseChildren: true,
		// By default (no Run/RunE in parent command) for typos in subcommands, cobra displays the help of parent command but exit(0) !
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			_ = cmd.Help()
			return dockercli.StatusError{
				StatusCode: metrics.CommandSyntaxFailure.ExitCode,
				Status:     fmt.Sprintf("unknown docker command: %q", "compose "+args[0]),
			}
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.HasParent() {
				parent := cmd.Root()
				if !parent.HasParent() {
					return nil
				}
				parentPrerun := parent.PersistentPreRunE
				if parentPrerun != nil {
					err := parentPrerun(cmd, args)
					if err != nil {
						return err
					}
				}
			}
			if noAnsi {
				if ansi != "auto" {
					return errors.New(`cannot specify DEPRECATED "--no-ansi" and "--ansi". Please use only "--ansi"`)
				}
				ansi = "never"
				fmt.Fprint(os.Stderr, aec.Apply("option '--no-ansi' is DEPRECATED ! Please use '--ansi' instead.\n", aec.RedF))
			}
			formatter.SetANSIMode(ansi)
			if opts.WorkDir != "" {
				if opts.ProjectDir != "" {
					return errors.New(`cannot specify DEPRECATED "--workdir" and "--project-directory". Please use only "--project-directory" instead`)
				}
				opts.ProjectDir = opts.WorkDir
				fmt.Fprint(os.Stderr, aec.Apply("option '--workdir' is DEPRECATED at root level! Please use '--project-directory' instead.\n", aec.RedF))
			}
			if contextType == store.DefaultContextType || contextType == store.LocalContextType {
				Warning = "The new 'docker compose' command is currently experimental. " +
					"To provide feedback or request new features please open issues at https://github.com/docker/compose-cli"
			}
			return nil
		},
	}

	command.AddCommand(
		upCommand(&opts, contextType, backend),
		downCommand(&opts, contextType, backend),
		startCommand(&opts, backend),
		restartCommand(&opts, backend),
		stopCommand(&opts, backend),
		psCommand(&opts, backend),
		listCommand(contextType, backend),
		logsCommand(&opts, contextType, backend),
		convertCommand(&opts, backend),
		killCommand(&opts, backend),
		runCommand(&opts, backend),
		removeCommand(&opts, backend),
		execCommand(&opts, backend),
		pauseCommand(&opts, backend),
		unpauseCommand(&opts, backend),
		topCommand(&opts, backend),
		eventsCommand(&opts, backend),
		portCommand(&opts, backend),
		imagesCommand(&opts, backend),
		versionCommand(),
	)

	if contextType == store.LocalContextType || contextType == store.DefaultContextType {
		command.AddCommand(
			buildCommand(&opts, backend),
			pushCommand(&opts, backend),
			pullCommand(&opts, backend),
			createCommand(&opts, backend),
			copyCommand(&opts, backend),
		)
	}
	command.Flags().SetInterspersed(false)
	opts.addProjectFlags(command.Flags())
	command.Flags().StringVar(&ansi, "ansi", "auto", `Control when to print ANSI control characters ("never"|"always"|"auto")`)
	command.Flags().BoolVar(&noAnsi, "no-ansi", false, `Do not print ANSI control characters (DEPRECATED)`)
	command.Flags().MarkHidden("no-ansi") //nolint:errcheck
	return command
}
