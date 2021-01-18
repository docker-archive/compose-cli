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

package lambdas

import (
	"fmt"

	"github.com/docker/compose-cli/api/lambdas"

	"github.com/compose-spec/compose-go/types"
)

func TransformForLambdas(project *types.Project) {
	queues := lambdas.GetQueues(project)
	if len(queues) == 0 {
		return
	}

	project.Networks["internal.events"] = types.NetworkConfig{
		Name: fmt.Sprintf("%s_internal.events", project.Name),
	}

	project.Services = append(project.Services, types.ServiceConfig{
		Name:     "internal.queues",
		Image:    "nats",
		Networks: map[string]*types.ServiceNetworkConfig{"internal.events": nil},
		Ports: []types.ServicePortConfig{
			{
				Target: 4222,
			},
			{
				Target: 8222,
			},
		},
	})
}
