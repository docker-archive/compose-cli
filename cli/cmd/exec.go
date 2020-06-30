/*
   Copyright 2020 Docker, Inc.

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
	"os/signal"
	"strings"

	"github.com/containerd/console"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/docker/api/client"
)

type execOpts struct {
	Tty         bool
	Interactive bool
	Command     string
}

// ExecCommand runs a command in a running container
func ExecCommand() *cobra.Command {
	var opts execOpts
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Run a command in a running container",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExec(cmd.Context(), opts, args[0], strings.Join(args[1:], " "))
		},
	}

	cmd.Flags().BoolVarP(&opts.Tty, "tty", "t", false, "Allocate a pseudo-TTY")
	cmd.Flags().BoolVarP(&opts.Interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	cmd.Flags().StringVar(&opts.Command, "command", "/bin/sh", "Shell used to exec the commands.")

	return cmd
}

func runExec(ctx context.Context, opts execOpts, name string, command string) error {
	c, err := client.New(ctx)
	if err != nil {
		return errors.Wrap(err, "cannot connect to backend")
	}

	con := console.Current()

	if opts.Tty {
		if err := con.SetRaw(); err != nil {
			return err
		}
		defer func() {
			if err := con.Reset(); err != nil {
				fmt.Println("Unable to close the console")
			}
		}()
	}

	suffixCommand := ""
	if !opts.Interactive {
		// suffixCommand = "\nexit\n"
		suffixCommand = " && exit\n"
	}
	rCon := &ComposedReader{
		Pre: strings.NewReader(command + suffixCommand),
	}

	if opts.Interactive {
		rCon.R = con
	}

	ctx, cancel := context.WithCancel(ctx)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		cancel()
		os.Exit(0)
	}()

	// for i, c := range opts.Command {
	// 	fmt.Printf("%d opts.Command: %d -> %s\n", i, c, string(opts.Command[i]))
	// }

	return c.ContainerService().Exec(ctx, name, opts.Command, rCon, con)
}

// ComposedReader provides a prepended Reader
type ComposedReader struct {
	Pre io.Reader
	R   io.Reader
}

func (c *ComposedReader) Read(p []byte) (int, error) {
	n, err := c.Pre.Read(p)
	if err == io.EOF {
		if c.R == nil {
			return n, err
		}
		return c.R.Read(p)
	}
	return n, err
}
