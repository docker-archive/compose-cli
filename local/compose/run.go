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
	"fmt"
	"io"
	"os"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose-cli/api/compose"
	convert "github.com/docker/compose-cli/local/moby"
	apitypes "github.com/docker/docker/api/types"
	moby "github.com/docker/docker/pkg/stringid"
)

func (s *composeService) CreateOneOffContainer(ctx context.Context, project *types.Project, opts compose.RunOptions) (string, error) {
	originalServices := project.Services
	dependencies := []types.ServiceConfig{}
	var requestedService types.ServiceConfig
	for _, service := range originalServices {
		if service.Name != opts.Name {
			dependencies = append(dependencies, service)
		} else {
			requestedService = service
		}
	}
	project.Services = types.Services(dependencies)
	if err := s.Create(ctx, project); err != nil {
		return "", err
	}
	if err := s.Start(ctx, project, nil); err != nil {
		return "", err
	}

	project.Services = originalServices
	updateOneOffServiceConfig(&requestedService, project.Name, opts)

	if err := s.waitDependencies(ctx, project, requestedService); err != nil {
		return "", err
	}
	err := s.createContainer(ctx, project, requestedService, requestedService.ContainerName, 1)
	if err != nil {
		return "", err
	}

	return requestedService.ContainerName, err
}

func (s *composeService) Run(ctx context.Context, container string, detach bool) error {
	if detach {
		return s.apiClient.ContainerStart(ctx, container, apitypes.ContainerStartOptions{})
	}

	cnx, err := s.apiClient.ContainerAttach(ctx, container, apitypes.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return err
	}
	defer cnx.Close()

	stdout := convert.ContainerStdout{HijackedResponse: cnx}
	stdin := convert.ContainerStdin{HijackedResponse: cnx}

	readChannel := make(chan error, 10)
	writeChannel := make(chan error, 10)

	go func() {
		_, err := io.Copy(os.Stdout, cnx.Reader)
		readChannel <- err
	}()

	go func() {
		_, err := io.Copy(stdin, os.Stdin)
		writeChannel <- err
	}()

	go func() {
		<-ctx.Done()
		stdout.Close() //nolint:errcheck
		stdin.Close()  //nolint:errcheck
	}()

	// start container
	err = s.apiClient.ContainerStart(ctx, container, apitypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	for {
		select {
		case err := <-readChannel:
			return err
		case err := <-writeChannel:
			return err
		}
	}
}

func updateOneOffServiceConfig(service *types.ServiceConfig, projectName string, opts compose.RunOptions) {
	if len(opts.Command) > 0 {
		service.Command = opts.Command
	}
	slug := moby.GenerateRandomID()
	service.Scale = 1
	service.ContainerName = fmt.Sprintf("%s_%s_run_%s", projectName, service.Name, moby.TruncateID(slug))
	service.Labels = types.Labels{
		slugLabel:   slug,
		oneoffLabel: "True",
	}
	service.Tty = true
	service.StdinOpen = true
}