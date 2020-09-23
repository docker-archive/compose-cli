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

package volume

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/api/client"
	"github.com/docker/compose-cli/api/volumes"
	"github.com/docker/compose-cli/errdefs"
	"github.com/docker/compose-cli/formatter"
)

type listVolumeOpts struct {
	format string
}

func listVolume() *cobra.Command {
	var opts listVolumeOpts
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "list available volumes in context.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.New(cmd.Context())
			if err != nil {
				return err
			}
			vols, err := c.VolumeService().List(cmd.Context())
			if err != nil {
				return err
			}
			return printList(opts.format, os.Stdout, vols)
		},
	}
	cmd.Flags().StringVar(&opts.format, "format", formatter.PRETTY, "Format the output. Values: [json | pretty]")
	return cmd
}

func printList(format string, out io.Writer, volumes []volumes.Volume) error {
	var err error
	switch strings.ToLower(format) {
	case formatter.JSON:
		out, err := formatter.ToStandardJSON(volumes)
		if err != nil {
			return err
		}
		fmt.Println(out)
	case formatter.PRETTY:
		printSection(out, func(w io.Writer) {
			for _, vol := range volumes {
				_, _ = fmt.Fprintf(w, "%s\t%s\n", vol.ID, vol.Description)
			}
		}, "ID", "DESCRIPTION")
	default:
		err = errors.Wrapf(errdefs.ErrParsingFailed, "format value %q could not be parsed", format)
	}
	return err
}

func printSection(out io.Writer, printer func(io.Writer), headers ...string) {
	w := tabwriter.NewWriter(out, 20, 1, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
	printer(w)
	_ = w.Flush()
}
