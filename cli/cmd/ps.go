package cmd

import (
	"encoding/json"
	client2 "github.com/docker/api/client"
	containersV1 "github.com/docker/api/containers/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var PsCommand = cobra.Command{
	Use:   "ps",
	Short: "List containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// get our current context
		ctx = current(ctx)

		client, err := client2.New("unix:///tmp/containersv1",100*time.Millisecond)
		if err != nil {
			return errors.Wrap(err, "cannot connect to backend")
		}
		defer client.Close()

		containersList, err := client.ContainersClient.List(ctx, &containersV1.ListRequest{})
		if err != nil {
			return errors.Wrap(err, "fetch backend information")
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", " ")
		return enc.Encode(containersList)
	},
}
