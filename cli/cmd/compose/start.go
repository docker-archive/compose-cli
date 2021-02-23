/*
   Copyright 2021 Docker Compose CLI authors

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

	"github.com/docker/compose-cli/api/client"
	"github.com/docker/compose-cli/api/compose"
	"github.com/docker/compose-cli/api/progress"

	"github.com/spf13/cobra"
)

type startOptions struct {
	*projectOptions
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

	_, err = progress.Run(ctx, func(ctx context.Context) (string, error) {
		return "", c.ComposeService().Start(ctx, project, compose.StartOptions{})
	})
	return err
}
