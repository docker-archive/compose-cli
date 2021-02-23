// +build kube

/*
   Copyright 2021 Docker Compose CLI authors

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
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/api/errdefs"
	"github.com/docker/compose-cli/kube"
)

func init() {
	extraCommands = append(extraCommands, createKubeCommand)
	extraHelp = append(extraHelp, `
Create a Kubernetes context:
$ docker context create k8s CONTEXT [flags]
(see docker context create k8s --help)
`)
}

func createKubeCommand() *cobra.Command {
	var opts kube.ContextParams
	cmd := &cobra.Command{
		Use:   "kubernetes CONTEXT [flags]",
		Short: "Create context for a Kubernetes Cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateKube(cmd.Context(), args[0], opts)
		},
	}

	addDescriptionFlag(cmd, &opts.Description)
	cmd.Flags().StringVar(&opts.KubeConfigPath, "kubeconfig", "", "The endpoint of the Kubernetes manager")
	cmd.Flags().StringVar(&opts.KubeContextName, "kubecontext", "", "The name of the context to use in kubeconfig")
	cmd.Flags().BoolVar(&opts.FromEnvironment, "from-env", false, "Get endpoint and creds from env vars")
	return cmd
}

func runCreateKube(ctx context.Context, contextName string, opts kube.ContextParams) error {
	if contextExists(ctx, contextName) {
		return errors.Wrapf(errdefs.ErrAlreadyExists, "context %q", contextName)
	}

	contextData, description, err := opts.CreateContextData()
	if err != nil {
		return err
	}
	return createDockerContext(ctx, contextName, store.KubeContextType, description, contextData)
}
