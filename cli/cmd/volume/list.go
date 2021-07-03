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

	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/v2/api/client"
	"github.com/docker/compose-cli/v2/api/volumes"
	"github.com/docker/compose-cli/v2/cli/formatter"
)

type listVolumeOpts struct {
	format string
	quiet  bool
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
			if opts.quiet {
				for _, v := range vols {
					fmt.Println(v.ID)
				}
				return nil
			}
			view := viewFromVolumeList(vols)
			return formatter.Print(view, opts.format, os.Stdout, func(w io.Writer) {
				for _, vol := range view {
					_, _ = fmt.Fprintf(w, "%s\t%s\n", vol.ID, vol.Description)
				}
			}, "ID", "DESCRIPTION")
		},
	}
	cmd.Flags().StringVar(&opts.format, "format", formatter.PRETTY, "Format the output. Values: [pretty | json]. (Default: pretty)")
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "Only display IDs")
	return cmd
}

type volumeView struct {
	ID          string
	Description string
}

func viewFromVolumeList(volumeList []volumes.Volume) []volumeView {
	retList := make([]volumeView, len(volumeList))
	for i, v := range volumeList {
		retList[i] = volumeView{
			ID:          v.ID,
			Description: v.Description,
		}
	}
	return retList
}
