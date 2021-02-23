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

package compose

import (
	"context"

	"github.com/docker/compose-cli/api/compose"
	"github.com/docker/compose-cli/api/progress"

	"github.com/compose-spec/compose-go/types"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func (s *composeService) Stop(ctx context.Context, project *types.Project, options compose.StopOptions) error {
	w := progress.ContextWriter(ctx)
	var containers Containers
	containers, err := s.apiClient.ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(projectFilter(project.Name)),
		All:     true,
	})
	if err != nil {
		return err
	}

	containers = containers.filter(isService(project.ServiceNames()...))

	return InReverseDependencyOrder(ctx, project, func(c context.Context, service types.ServiceConfig) error {
		return s.stopContainers(ctx, w, containers.filter(isService(service.Name)), options.Timeout)
	})
}
