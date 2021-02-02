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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/compose-cli/api/compose"

	"github.com/compose-spec/compose-go/types"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sanathkr/go-yaml"

	errdefs2 "github.com/docker/compose-cli/api/errdefs"
)

// NewComposeService create a local implementation of the compose.Service API
func NewComposeService(apiClient *client.Client) compose.Service {
	return &composeService{apiClient: apiClient}
}

type composeService struct {
	apiClient *client.Client
}

func (s *composeService) Up(ctx context.Context, project *types.Project, options compose.UpOptions) error {
	return errdefs2.ErrNotImplemented
}

func getCanonicalContainerName(c moby.Container) string {
	// Names return container canonical name /foo  + link aliases /linked_by/foo
	for _, name := range c.Names {
		if strings.LastIndex(name, "/") == 0 {
			return name[1:]
		}
	}
	return c.Names[0][1:]
}

func (s *composeService) Convert(ctx context.Context, project *types.Project, options compose.ConvertOptions) ([]byte, error) {
	switch options.Format {
	case "json":
		return json.MarshalIndent(project, "", "  ")
	case "yaml":
		return yaml.Marshal(project)
	default:
		return nil, fmt.Errorf("unsupported format %q", options)
	}
}

// ProjectConfigView is the project view representation of a compose file
type ProjectConfigView struct {
	Services types.Services `json:"services"`
	Networks types.Networks `yaml:",omitempty" json:"networks,omitempty"`
	Volumes  types.Volumes  `yaml:",omitempty" json:"volumes,omitempty"`
	Secrets  types.Secrets  `yaml:",omitempty" json:"secrets,omitempty"`
	Configs  types.Configs  `yaml:",omitempty" json:"configs,omitempty"`
}

func (pv *ProjectConfigView) fromProject(project *types.Project) {
	pv.Services = project.Services
	pv.Networks = project.Networks
	pv.Volumes = project.Volumes
	pv.Secrets = project.Secrets
	pv.Configs = project.Configs
}

func (s *composeService) Config(ctx context.Context, project *types.Project, options compose.ConfigOptions) ([]byte, error) {
	var pv ProjectConfigView
	pv.fromProject(project)
	switch options.Format {
	case "json":
		return json.MarshalIndent(pv, "", "  ")
	case "yaml":
		return yaml.Marshal(pv)
	default:
		return nil, fmt.Errorf("unsupported format %q", options.Format)
	}
}
