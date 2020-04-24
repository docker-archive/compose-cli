package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/docker/api/util"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var psCommand = cli.Command{
	Name:  "ps",
	Usage: "list containers",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "all,a",
			Usage: "display all containers",
		},
	},
	Action: func(clix *cli.Context) error {
		all := clix.Bool("all")

		// return information for the current context
		ctx, cancel := util.NewSigContext()
		defer cancel()

		// get our current context
		ctx = current(ctx)

		client, err := connect(ctx)
		if err != nil {
			return errors.Wrap(err, "cannot connect to backend")
		}
		defer client.Close()

		containers, err := client.List(ctx)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
		const tfmt = "%s\t%s\n"
		fmt.Fprint(w, "ID\tSTATUS\n")

		for _, c := range containers {
			if all || c.Status != "stopped" {
				fmt.Fprintf(w, tfmt,
					c.ID,
					c.Status,
				)
			}
		}

		w.Flush()

		return nil
	},
}
