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
	"fmt"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
	"github.com/morikuni/aec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/docker/compose-cli/api/context/store"
)

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

func (o *projectOptions) toProject(services []string) (*types.Project, error) {
	options, err := o.toProjectOptions()
	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(options)
	if err != nil {
		return nil, err
	}

	if len(services) != 0 {
		s, err := project.GetServices(services...)
		if err != nil {
			return nil, err
		}
		o.Profiles = append(o.Profiles, s.GetProfiles()...)
	}

	project.ApplyProfiles(o.Profiles)

	err = project.ForServices(services)
	return project, err
}

func (o *projectOptions) toProjectOptions() (*cli.ProjectOptions, error) {
	return cli.NewProjectOptions(o.ConfigPaths,
		cli.WithEnvFile(o.EnvFile),
		cli.WithDotEnv,
		cli.WithOsEnv,
		cli.WithWorkingDirectory(o.ProjectDir),
		cli.WithName(o.ProjectName))
}

// Command returns the compose command with its child commands
func Command(contextType string) *cobra.Command {
	opts := projectOptions{}
	command := &cobra.Command{
		Short:            "Docker Compose",
		Use:              "compose",
		TraverseChildren: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.WorkDir != "" {
				if opts.ProjectDir != "" {
					return errors.New(aec.Apply(`cannot specify DEPRECATED "--workdir" and "--project-directory". Please use only "--project-directory" instead.`, aec.RedF))
				}
				opts.ProjectDir = opts.WorkDir
				fmt.Println(aec.Apply(`option "--workdir" is DEPRECATED at root level! Please use "--project-directory" instead.`, aec.RedF))
			}
			if contextType == store.DefaultContextType || contextType == store.LocalContextType {
				Warning = "The new 'docker compose' command is currently experimental. " +
					"To provide feedback or request new features please open issues at https://github.com/docker/compose-cli"
			}
			return nil
		},
	}

	command.AddCommand(
		upCommand(&opts, contextType),
		downCommand(&opts, contextType),
		startCommand(&opts),
		stopCommand(&opts),
		psCommand(&opts),
		listCommand(contextType),
		logsCommand(&opts, contextType),
		convertCommand(&opts),
		killCommand(&opts),
		runCommand(&opts),
		removeCommand(&opts),
		execCommand(&opts),
		pauseCommand(&opts),
		unpauseCommand(&opts),
		completionCommand(),
	)

	if contextType == store.LocalContextType || contextType == store.DefaultContextType {
		command.AddCommand(
			buildCommand(&opts),
			pushCommand(&opts),
			pullCommand(&opts),
			createCommand(&opts),
		)
	}
	command.Flags().SetInterspersed(false)
	opts.addProjectFlags(command.Flags())
	return command
}
