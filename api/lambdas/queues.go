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
	"github.com/compose-spec/compose-go/types"
	"github.com/mitchellh/mapstructure"
)

type Queue struct {
	External bool   `yaml:",omitempty" json:"external,omitempty"`
	Name     string `yaml:",omitempty" json:"name,omitempty"`
}

type Queues map[string]Queue

// TODO to be moved to compose-go loader.Load once compose-spec has approved `queues` design
func LoadQueues(project *types.Project) error {
	x, ok := project.Extensions["x-queues"]
	if !ok {
		return nil
	}

	queues := Queues{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &queues,
	})
	if err != nil {
		return err
	}
	err = decoder.Decode(x)
	if err != nil {
		return err
	}
	project.Extensions["x-queues"] = queues
	return nil
}

// GetQueues returns project's queues.
// TODO replace use of this with direct access `project.Queues` once compose-spec approvec `queues` concept
func GetQueues(project *types.Project) Queues {
	x, ok := project.Extensions["x-queues"]
	if !ok {
		return nil
	}
	return x.(Queues)
}
