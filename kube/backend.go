// +build kube

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

package kube

import (
	"github.com/docker/compose-cli/v2/api/backend"
	"github.com/docker/compose-cli/v2/api/cloud"
	"github.com/docker/compose-cli/v2/api/containers"
	"github.com/docker/compose-cli/v2/api/context/store"
	"github.com/docker/compose-cli/v2/api/resources"
	"github.com/docker/compose-cli/v2/api/secrets"
	"github.com/docker/compose-cli/v2/api/volumes"
	"github.com/docker/compose-cli/v2/pkg/api"
)

const backendType = store.KubeContextType

type kubeAPIService struct {
	composeService api.Service
}

func init() {
	backend.Register(backendType, backendType, service, cloud.NotImplementedCloudService)
}

func service() (backend.Service, error) {
	s, err := NewComposeService()
	if err != nil {
		return nil, err
	}
	return &kubeAPIService{
		composeService: s,
	}, nil
}

func (s *kubeAPIService) ContainerService() containers.Service {
	return nil
}

func (s *kubeAPIService) ComposeService() api.Service {
	return s.composeService
}

func (s *kubeAPIService) SecretsService() secrets.Service {
	return nil
}

func (s *kubeAPIService) VolumeService() volumes.Service {
	return nil
}

func (s *kubeAPIService) ResourceService() resources.Service {
	return nil
}
