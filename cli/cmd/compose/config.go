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

	"github.com/docker/compose-cli/api/compose"

	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/api/client"
)

type configOptions struct {
	*projectOptions
	Format string
	Quiet  bool
}

func configCommand(p *projectOptions) *cobra.Command {
	opts := configOptions{
		projectOptions: p,
	}
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Validate and view the configuration of the intended deployment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(cmd.Context(), opts)
		},
	}

	flags := configCmd.Flags()
	flags.StringVar(&opts.Format, "format", "", "Format the output. Values: [yaml | json]")

	return configCmd
}

func runConfig(ctx context.Context, opts configOptions) error {
	var out []byte
	c, err := client.NewWithDefaultLocalBackend(ctx)
	if err != nil {
		return err
	}
	project, err := opts.toProject(nil)
	if err != nil {
		return err
	}
	if opts.Quiet {
		return nil
	}
	out, err = c.ComposeService().Config(ctx, project, compose.ConfigOptions{
		Format: opts.Format,
		Quiet:  opts.Quiet,
	})
	if err != nil {
		return err
	}
	fmt.Print(string(out))

	return nil
}
