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

	"github.com/docker/compose-cli/pkg/api"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/streams"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/stringid"
	"github.com/moby/term"
)

func (s *composeService) RunOneOffContainer(ctx context.Context, project *types.Project, opts api.RunOptions) (int, error) {
	observedState, err := s.getContainers(ctx, project.Name, oneOffInclude, true)
	if err != nil {
		return 0, err
	}

	containerID, err := s.prepareRun(ctx, project, observedState, opts)
	if err != nil {
		return 0, err
	}

	if opts.Detach {
		err := s.apiClient.ContainerStart(ctx, containerID, moby.ContainerStartOptions{})
		if err != nil {
			return 0, err
		}
		fmt.Fprintln(opts.Stdout, containerID)
		return 0, nil
	}

	return s.runInteractive(ctx, containerID, opts)
}

func (s *composeService) runInteractive(ctx context.Context, containerID string, opts api.RunOptions) (int, error) {
	r, err := s.getEscapeKeyProxy(opts.Stdin)
	if err != nil {
		return 0, err
	}

	stdin, stdout, err := s.getContainerStreams(ctx, containerID)
	if err != nil {
		return 0, err
	}

	in := streams.NewIn(opts.Stdin)
	if in.IsTerminal() {
		state, err := term.SetRawTerminal(in.FD())
		if err != nil {
			return 0, err
		}
		defer term.RestoreTerminal(in.FD(), state) //nolint:errcheck
	}

	outputDone := make(chan error)
	inputDone := make(chan error)

	go func() {
		if opts.Tty {
			_, err := io.Copy(opts.Stdout, stdout) //nolint:errcheck
			outputDone <- err
		} else {
			_, err := stdcopy.StdCopy(opts.Stdout, opts.Stderr, stdout) //nolint:errcheck
			outputDone <- err
		}
		stdout.Close() //nolint:errcheck
	}()

	go func() {
		_, err := io.Copy(stdin, r)
		inputDone <- err
		stdin.Close() //nolint:errcheck
	}()

	err = s.apiClient.ContainerStart(ctx, containerID, moby.ContainerStartOptions{})
	if err != nil {
		return 0, err
	}

	s.monitorTTySize(ctx, containerID, s.apiClient.ContainerResize)

	for {
		select {
		case err := <-outputDone:
			if err != nil {
				return 0, err
			}
			inspect, err := s.apiClient.ContainerInspect(ctx, containerID)
			if err != nil {
				return 0, err
			}
			exitCode := 0
			if inspect.State != nil {
				exitCode = inspect.State.ExitCode
			}
			return exitCode, nil
		case err := <-inputDone:
			if _, ok := err.(term.EscapeError); ok {
				return 0, nil
			}
			if err != nil {
				return 0, err
			}
			// Wait for output to complete streaming
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
}

func (s *composeService) prepareRun(ctx context.Context, project *types.Project, observedState Containers, opts api.RunOptions) (string, error) {
	service, err := project.GetService(opts.Service)
	if err != nil {
		return "", err
	}

	applyRunOptions(project, &service, opts)

	slug := stringid.GenerateRandomID()
	if service.ContainerName == "" {
		service.ContainerName = fmt.Sprintf("%s_%s_run_%s", project.Name, service.Name, stringid.TruncateID(slug))
	}
	service.Scale = 1
	service.StdinOpen = true
	service.Restart = ""
	if service.Deploy != nil {
		service.Deploy.RestartPolicy = nil
	}
	service.Labels = service.Labels.Add(api.SlugLabel, slug)
	service.Labels = service.Labels.Add(api.OneoffLabel, "True")

	if err := s.ensureImagesExists(ctx, project, observedState, false); err != nil { // all dependencies already checked, but might miss service img
		return "", err
	}
	if err := s.waitDependencies(ctx, project, service); err != nil {
		return "", err
	}
	created, err := s.createContainer(ctx, project, service, service.ContainerName, 1, opts.AutoRemove, opts.UseNetworkAliases)
	if err != nil {
		return "", err
	}
	containerID := created.ID
	return containerID, nil
}

func (s *composeService) getEscapeKeyProxy(r io.ReadCloser) (io.ReadCloser, error) {
	var escapeKeys = []byte{16, 17}
	if s.configFile.DetachKeys != "" {
		customEscapeKeys, err := term.ToBytes(s.configFile.DetachKeys)
		if err != nil {
			return nil, err
		}
		escapeKeys = customEscapeKeys
	}
	return ioutils.NewReadCloserWrapper(term.NewEscapeProxy(r, escapeKeys), r.Close), nil
}

func applyRunOptions(project *types.Project, service *types.ServiceConfig, opts api.RunOptions) {
	service.Tty = opts.Tty
	service.ContainerName = opts.Name

	if len(opts.Command) > 0 {
		service.Command = opts.Command
	}
	if len(opts.User) > 0 {
		service.User = opts.User
	}
	if len(opts.WorkingDir) > 0 {
		service.WorkingDir = opts.WorkingDir
	}
	if len(opts.Entrypoint) > 0 {
		service.Entrypoint = opts.Entrypoint
	}
	if len(opts.Environment) > 0 {
		env := types.NewMappingWithEquals(opts.Environment)
		projectEnv := env.Resolve(func(s string) (string, bool) {
			v, ok := project.Environment[s]
			return v, ok
		}).RemoveEmpty()
		service.Environment.OverrideBy(projectEnv)
	}
	for k, v := range opts.Labels {
		service.Labels = service.Labels.Add(k, v)
	}
}
