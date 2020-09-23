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

package context

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/cli/mobycli"
	apicontext "github.com/docker/compose-cli/context"
	"github.com/docker/compose-cli/context/store"
	"github.com/docker/compose-cli/errdefs"
	"github.com/docker/compose-cli/formatter"
)

type lsOpts struct {
	quiet  bool
	json   bool
	format string
}

func (o lsOpts) validate() error {
	if o.quiet && o.json {
		return errors.New(`cannot combine "quiet" and "json" options`)
	}
	return nil
}

func listCommand() *cobra.Command {
	var opts lsOpts
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List available contexts",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "Only show context names")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Format output as JSON")
	cmd.Flags().StringVar(&opts.format, "format", formatter.PRETTY, "Format the output. Values: [json | pretty]")

	return cmd
}

func runList(cmd *cobra.Command, opts lsOpts) error {
	err := opts.validate()
	if err != nil {
		return err
	}
	if opts.format != "" {
		mobycli.Exec(cmd.Root())
		return nil
	}

	ctx := cmd.Context()
	currentContext := apicontext.CurrentContext(ctx)
	s := store.ContextStore(ctx)
	contexts, err := s.List()
	if err != nil {
		return err
	}

	sort.Slice(contexts, func(i, j int) bool {
		return strings.Compare(contexts[i].Name, contexts[j].Name) == -1
	})

	if opts.quiet {
		for _, c := range contexts {
			fmt.Println(c.Name)
		}
		return nil
	}

	if opts.json {
		j, err := formatter.ToStandardJSON(contexts)
		if err != nil {
			return err
		}
		fmt.Println(j)
		return nil
	}

	return printContextLsFormatted(opts.format, currentContext, os.Stdout, contexts)
}

func printContextLsFormatted(format string, currContext string, out io.Writer, contexts []*store.DockerContext) error {
	var err error
	switch strings.ToLower(format) {
	case formatter.JSON:
		out, err := formatter.ToStandardJSON(contexts)
		if err != nil {
			return err
		}
		fmt.Println(out)
	case formatter.PRETTY:
		err = formatter.PrintPrettySection(out, func(w io.Writer) {
			for _, c := range contexts {
				contextName := c.Name
				if c.Name == currContext {
					contextName += " *"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					contextName,
					c.Type(),
					c.Metadata.Description,
					getEndpoint("docker", c.Endpoints),
					getEndpoint("kubernetes", c.Endpoints),
					c.Metadata.StackOrchestrator)
			}
		}, "NAME", "TYPE", "DESCRIPTION", "DOCKER ENDPOINT", "KUBERNETES ENDPOINT", "ORCHESTRATOR")

	default:
		err = errors.Wrapf(errdefs.ErrParsingFailed, "format value %q could not be parsed", format)
	}
	return err
}

func getEndpoint(name string, meta map[string]interface{}) string {
	endpoints, ok := meta[name]
	if !ok {
		return ""
	}
	data, ok := endpoints.(*store.Endpoint)
	if !ok {
		return ""
	}

	result := data.Host
	if data.DefaultNamespace != "" {
		result += fmt.Sprintf(" (%s)", data.DefaultNamespace)
	}

	return result
}
