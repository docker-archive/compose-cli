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
	"testing"
	"time"

	"github.com/docker/compose-cli/api/compose"
	"github.com/docker/compose-cli/local/mocks"
	moby "github.com/docker/docker/api/types"

	"github.com/compose-spec/compose-go/types"
	"github.com/golang/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestStopTimeout(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	api := mocks.NewMockAPIClient(mockCtrl)
	tested.apiClient = api

	ctx := context.Background()
	api.EXPECT().ContainerList(ctx, projectFilterListOpt(testProject)).Return(
		[]moby.Container{
			testContainer("service1", "123"),
			testContainer("service1", "456"),
			testContainer("service2", "789"),
		}, nil)

	timeout := time.Duration(2) * time.Second
	api.EXPECT().ContainerStop(ctx, "123", &timeout).Return(nil)
	api.EXPECT().ContainerStop(ctx, "456", &timeout).Return(nil)
	api.EXPECT().ContainerStop(ctx, "789", &timeout).Return(nil)

	err := tested.Stop(ctx, &types.Project{
		Name: testProject,
		Services: []types.ServiceConfig{
			{Name: "service1"},
			{Name: "service2"},
		},
	}, compose.StopOptions{
		Timeout: &timeout,
	})
	assert.NilError(t, err)
}
