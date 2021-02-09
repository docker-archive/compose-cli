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
	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/api/client"
	"github.com/docker/compose-cli/api/compose"
	"github.com/docker/compose-cli/api/progress"
)

type startOptions struct {
	*projectOptions
	Detach bool
}

func startCommand(p *projectOptions) *cobra.Command {
	opts := startOptions{
		projectOptions: p,
	}
	startCmd := &cobra.Command{
		Use:   "start [SERVICE...]",
		Short: "Start services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd.Context(), opts, args)
		},
	}

	startCmd.Flags().BoolVarP(&opts.Detach, "detach", "d", false, "Detached mode: Run containers in the background")
	return startCmd
}

func runStart(ctx context.Context, opts startOptions, services []string) error {
	c, err := client.NewWithDefaultLocalBackend(ctx)
	if err != nil {
		return err
	}

	project, err := opts.toProject(services)
	if err != nil {
		return err
	}

	if opts.Detach {
		_, err = progress.Run(ctx, func(ctx context.Context) (string, error) {
			return "", c.ComposeService().Start(ctx, project, compose.StartOptions{})
		})
		return err
	}

	queue := make(chan compose.ContainerEvent)
	printer := Printer{
		queue: queue,
	}
	err = c.ComposeService().Start(ctx, project, compose.StartOptions{
		Attach: queue,
	})
	if err != nil {
		return err
	}

	_, err = printer.Run(ctx, false, func() error {
		ctx := context.Background()
		_, err := progress.Run(ctx, func(ctx context.Context) (string, error) {
			return "", c.ComposeService().Stop(ctx, project)
		})
		return err
	})
	return err
}
