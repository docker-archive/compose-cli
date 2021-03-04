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

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose-cli/api/backend"
	"github.com/docker/compose-cli/api/config"
	apicontext "github.com/docker/compose-cli/api/context"
	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/cli/cmd/compose"
	cliconfig "github.com/docker/compose-cli/cli/config"
	cliopts "github.com/docker/compose-cli/cli/options"
	"github.com/docker/compose-cli/internal"
	"github.com/docker/compose-cli/local"
)

func main() {
	var opts cliopts.GlobalOpts
	root := &cobra.Command{
		Use: "docker",
	}
	flags := root.Flags()
	opts.InstallFlags(flags)
	opts.AddConfigFlags(flags)
	flags.SetInterspersed(false)
	// populate the opts with the global flags
	err := flags.Parse(os.Args[1:]) //nolint: errcheck
	if err != nil {
		log.Fatal(err)
	}

	service, err := local.GetLocalBackend(opts.Config, opts)
	if err != nil {
		log.Fatal(err)
	}

	if opts.Config == "" {
		log.Fatal(fmt.Errorf("config path cannot be empty"))
	}
	configDir := opts.Config
	config.WithDir(configDir)

	currentContext := cliconfig.GetCurrentContext(opts.Context, opts.Config, opts.Hosts)
	apicontext.WithCurrentContext(currentContext)

	s, err := store.New(configDir)
	if err != nil {
		log.Fatal(err)
	}
	store.WithContextStore(s)

	backend.WithBackend(service)

	plugin.Run(func(dockerCli command.Cli) *cobra.Command {
		return compose.Command(store.DefaultContextType)
	},
		manager.Metadata{
			SchemaVersion: "0.1.0",
			Vendor:        "Docker Inc.",
			Version:       strings.TrimPrefix(internal.Version, "v"),
		})
}
