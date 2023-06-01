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

package ecs

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/compose-cli/api/backend"
	"github.com/docker/compose-cli/api/cloud"
	"github.com/docker/compose-cli/api/containers"
	apicontext "github.com/docker/compose-cli/api/context"
	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/api/resources"
	"github.com/docker/compose-cli/api/secrets"
	"github.com/docker/compose-cli/api/volumes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/docker/compose/v2/pkg/api"
)

const backendType = store.EcsContextType

// ContextParams options for creating AWS context
type ContextParams struct {
	Name         string
	Description  string
	AccessKey    string
	SecretKey    string
	Profile      string
	Region       string
	CredsFromEnv bool
}

func (c ContextParams) haveRequiredEnvVars() bool {
	if c.Profile != "" {
		return true
	}
	if c.AccessKey != "" && c.SecretKey != "" {
		return true
	}
	return false
}

func init() {
	backend.Register(backendType, backendType, service, getCloudService)
}

func service() (backend.Service, error) {
	fmt.Fprintln(os.Stderr, "Cloud integration is DEPRECATED. Read more on https://docs.docker.com/cloud/ecs-integration/")
	contextStore := store.Instance()
	currentContext := apicontext.Current()
	var ecsContext store.EcsContext

	if err := contextStore.GetEndpoint(currentContext, &ecsContext); err != nil {
		return nil, err
	}

	return getEcsAPIService(ecsContext)
}

func getEcsAPIService(ecsCtx store.EcsContext) (*ecsAPIService, error) {
	region := ""
	profile := ecsCtx.Profile

	if ecsCtx.CredentialsFromEnv {
		env := getEnvVars()
		if !env.haveRequiredEnvVars() {
			return nil, fmt.Errorf("context requires credentials to be passed as environment variables")
		}
		profile = env.Profile
		region = env.Region
	}

	if region == "" {
		r, err := getRegion(profile)
		if err != nil {
			return nil, err
		}
		region = r
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:           profile,
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String(region),
		},
	})
	if err != nil {
		return nil, err
	}

	sdk := newSDK(sess)
	return &ecsAPIService{
		ctx:    ecsCtx,
		Region: region,
		aws:    sdk,
	}, nil
}

type ecsAPIService struct {
	ctx    store.EcsContext
	Region string
	aws    API
}

func (b *ecsAPIService) ContainerService() containers.Service {
	return nil
}

func (b *ecsAPIService) ComposeService() api.Service {
	return b
}

func (b *ecsAPIService) SecretsService() secrets.Service {
	return b
}

func (b *ecsAPIService) VolumeService() volumes.Service {
	return ecsVolumeService{backend: b}
}

func (b *ecsAPIService) ResourceService() resources.Service {
	return nil
}

func getCloudService() (cloud.Service, error) {
	return ecsCloudService{}, nil
}

type ecsCloudService struct {
}

func (a ecsCloudService) Login(ctx context.Context, params interface{}) error {
	return api.ErrNotImplemented
}

func (a ecsCloudService) Logout(ctx context.Context) error {
	return api.ErrNotImplemented
}

func (a ecsCloudService) CreateContextData(ctx context.Context, params interface{}) (interface{}, string, error) {
	fmt.Fprintln(os.Stderr, "Cloud integration is DEPRECATED. Read more on https://docs.docker.com/cloud/ecs-integration/")
	contextHelper := newContextCreateHelper()
	createOpts := params.(ContextParams)
	return contextHelper.createContextData(ctx, createOpts)
}
