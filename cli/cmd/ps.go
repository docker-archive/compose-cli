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

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/api/client"
	"github.com/docker/compose-cli/api/containers"
	"github.com/docker/compose-cli/errdefs"
	formatter2 "github.com/docker/compose-cli/formatter"
	"github.com/docker/compose-cli/utils/formatter"
)

type psOpts struct {
	all    bool
	quiet  bool
	format string
}

// PsCommand lists containers
func PsCommand() *cobra.Command {
	var opts psOpts
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPs(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "Only display IDs")
	cmd.Flags().BoolVarP(&opts.all, "all", "a", false, "Show all containers (default shows just running)")
	cmd.Flags().StringVar(&opts.format, "format", formatter2.PRETTY, "Format the output. Values: [json | pretty]")

	return cmd
}

func runPs(ctx context.Context, opts psOpts) error {
	c, err := client.New(ctx)
	if err != nil {
		return errors.Wrap(err, "cannot connect to backend")
	}

	containerList, err := c.ContainerService().List(ctx, opts.all)
	if err != nil {
		return errors.Wrap(err, "fetch containerList")
	}

	if opts.quiet {
		for _, c := range containerList {
			fmt.Println(c.ID)
		}
		return nil
	}

	return printPsFormatted(opts.format, os.Stdout, containerList)
}

func printPsFormatted(format string, out io.Writer, containers []containers.Container) error {
	var err error
	switch strings.ToLower(format) {
	case formatter2.JSON:
		out, err := formatter2.ToStandardJSON(containers)
		if err != nil {
			return err
		}
		fmt.Println(out)
	case formatter2.PRETTY:
		err = formatter2.PrintPrettySection(out, func(w io.Writer) {
			for _, c := range containers {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", c.ID, c.Image, c.Command, c.Status,
					strings.Join(formatter.PortsToStrings(c.Ports, fqdn(c)), ", "))
			}
		}, "CONTAINER ID", "IMAGE", "COMMAND", "STATUS", "PORTS")

	default:
		err = errors.Wrapf(errdefs.ErrParsingFailed, "format value %q could not be parsed", format)
	}
	return err
}

func fqdn(container containers.Container) string {
	fqdn := ""
	if container.Config != nil {
		fqdn = container.Config.FQDN
	}
	return fqdn
}
