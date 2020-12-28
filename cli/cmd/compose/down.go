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

	"github.com/docker/compose-cli/api/compose"

	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/api/client"
	"github.com/docker/compose-cli/progress"
)

func downCommand(composeOpts composeOptions) *cobra.Command {
	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop and remove containers, networks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDown(cmd.Context(), composeOpts)
		},
	}

	downCmd.Flags().StringVar(&composeOpts.WorkingDir, "workdir", "", "Work dir")

	return downCmd
}

func runDown(ctx context.Context, opts composeOptions) error {
	c, err := client.NewWithDefaultLocalBackend(ctx)
	if err != nil {
		return err
	}

	_, err = progress.Run(ctx, func(ctx context.Context) (string, error) {
		projectName, err := opts.toProjectName()
		if err != nil {
			return "", err
		}
		return projectName, c.ComposeService().Down(ctx, projectName, compose.DownOptions{
			RemoveOrphans: false,
		})
	})
	return err
}
