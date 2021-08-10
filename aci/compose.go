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

package aci

import (
	"context"
	"fmt"
	"net/http"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose-cli/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"

	"github.com/docker/compose-cli/aci/convert"
	"github.com/docker/compose-cli/aci/login"
	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/pkg/api"
	"github.com/docker/compose-cli/pkg/progress"
	"github.com/docker/compose-cli/utils/formatter"
)

type aciComposeService struct {
	ctx          store.AciContext
	storageLogin login.StorageLoginImpl
}

func newComposeService(ctx store.AciContext) aciComposeService {
	return aciComposeService{
		ctx:          ctx,
		storageLogin: login.StorageLoginImpl{AciContext: ctx},
	}
}

func (cs *aciComposeService) Build(ctx context.Context, project *types.Project, options api.BuildOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Push(ctx context.Context, project *types.Project, options api.PushOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Pull(ctx context.Context, project *types.Project, options api.PullOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Create(ctx context.Context, project *types.Project, opts api.CreateOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Start(ctx context.Context, project *types.Project, options api.StartOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Restart(ctx context.Context, project *types.Project, options api.RestartOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Stop(ctx context.Context, project *types.Project, options api.StopOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Pause(ctx context.Context, project string, options api.PauseOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) UnPause(ctx context.Context, project string, options api.PauseOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Copy(ctx context.Context, project *types.Project, options api.CopyOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Up(ctx context.Context, project *types.Project, options api.UpOptions) error {
	if err := checkUnSupportedUpOptions(options); err != nil {
		return err
	}
	return progress.Run(ctx, func(ctx context.Context) error {
		return cs.up(ctx, project)
	})
}

func checkUnSupportedUpOptions(o api.UpOptions) error {
	var errs *multierror.Error
	errs = utils.CheckUnsupported(errs, o.Start.CascadeStop, false, "abort-on-container-exit")
	errs = utils.CheckUnsupported(errs, o.Create.RecreateDependencies, "", "always-recreate-deps")
	errs = utils.CheckUnsupported(errs, len(o.Start.AttachTo), 0, "attach-dependencies")
	errs = utils.CheckUnsupported(errs, len(o.Start.ExitCodeFrom), 0, "exit-code-from")
	errs = utils.CheckUnsupported(errs, o.Create.Recreate, "", "force-recreate")
	errs = utils.CheckUnsupported(errs, o.Create.QuietPull, false, "quiet-pull")
	errs = utils.CheckUnsupported(errs, o.Create.RemoveOrphans, false, "remove-orphans")
	errs = utils.CheckUnsupported(errs, o.Create.Inherit, false, "renew-anon-volumes")
	errs = utils.CheckUnsupportedDurationPtr(errs, o.Create.Timeout, nil, "timeout")
	return errs.ErrorOrNil()
}

func (cs *aciComposeService) up(ctx context.Context, project *types.Project) error {
	logrus.Debugf("Up on project with name %q", project.Name)

	if err := autocreateFileshares(ctx, project); err != nil {
		return err
	}

	groupDefinition, err := convert.ToContainerGroup(ctx, cs.ctx, *project, cs.storageLogin)
	if err != nil {
		return err
	}

	addTag(&groupDefinition, composeContainerTag)
	return createOrUpdateACIContainers(ctx, cs.ctx, groupDefinition)
}

func (cs aciComposeService) warnKeepVolumeOnDown(ctx context.Context, projectName string) error {
	cgClient, err := login.NewContainerGroupsClient(cs.ctx.SubscriptionID)
	if err != nil {
		return err
	}
	cg, err := cgClient.Get(ctx, cs.ctx.ResourceGroup, projectName)
	if err != nil {
		return err
	}
	if cg.Volumes == nil {
		return nil
	}
	for _, v := range *cg.Volumes {
		if v.AzureFile == nil || v.AzureFile.StorageAccountName == nil || v.AzureFile.ShareName == nil {
			continue
		}
		fmt.Printf("WARNING: fileshare \"%s/%s\" will NOT be deleted. Use 'docker volume rm' if you want to delete this volume\n",
			*v.AzureFile.StorageAccountName, *v.AzureFile.ShareName)
	}
	return nil
}

func (cs *aciComposeService) Down(ctx context.Context, projectName string, options api.DownOptions) error {
	if err := checkUnSupportedDownOptions(options); err != nil {
		return err
	}
	return progress.Run(ctx, func(ctx context.Context) error {
		logrus.Debugf("Down on project with name %q", projectName)

		if err := cs.warnKeepVolumeOnDown(ctx, projectName); err != nil {
			return err
		}

		cg, err := deleteACIContainerGroup(ctx, cs.ctx, projectName)
		if err != nil {
			return err
		}
		if cg.IsHTTPStatus(http.StatusNoContent) {
			return api.ErrNotFound
		}

		return err
	})
}

func checkUnSupportedDownOptions(o api.DownOptions) error {
	var errs *multierror.Error
	errs = utils.CheckUnsupported(errs, o.Volumes, false, "volumes")
	errs = utils.CheckUnsupported(errs, o.Images, "", "images")
	return errs.ErrorOrNil()
}

func (cs *aciComposeService) Ps(ctx context.Context, projectName string, options api.PsOptions) ([]api.ContainerSummary, error) {
	if err := checkUnSupportedPsOptions(options); err != nil {
		return nil, err
	}

	groupsClient, err := login.NewContainerGroupsClient(cs.ctx.SubscriptionID)
	if err != nil {
		return nil, err
	}

	group, err := groupsClient.Get(ctx, cs.ctx.ResourceGroup, projectName)
	if err != nil {
		return nil, err
	}

	if group.Containers == nil || len(*group.Containers) == 0 {
		return nil, fmt.Errorf("no containers found in ACI container group %s", projectName)
	}

	res := []api.ContainerSummary{}
	for _, container := range *group.Containers {
		if isContainerVisible(container, group, false) {
			continue
		}
		var publishers []api.PortPublisher
		urls := formatter.PortsToStrings(convert.ToPorts(group.IPAddress, *container.Ports), convert.FQDN(group, cs.ctx.Location))
		for i, p := range *container.Ports {
			publishers = append(publishers, api.PortPublisher{
				URL:           urls[i],
				TargetPort:    int(*p.Port),
				PublishedPort: int(*p.Port),
				Protocol:      string(p.Protocol),
			})
		}
		id := getContainerID(group, container)
		res = append(res, api.ContainerSummary{
			ID:         id,
			Name:       id,
			Project:    projectName,
			Service:    *container.Name,
			State:      convert.GetStatus(container, group),
			Publishers: publishers,
		})
	}
	return res, nil
}

func checkUnSupportedPsOptions(o api.PsOptions) error {
	var errs *multierror.Error
	errs = utils.CheckUnsupported(errs, o.All, false, "all")
	return errs.ErrorOrNil()
}

func (cs *aciComposeService) List(ctx context.Context, opts api.ListOptions) ([]api.Stack, error) {
	if err := checkUnSupportedListOptions(opts); err != nil {
		return nil, err
	}

	containerGroups, err := getACIContainerGroups(ctx, cs.ctx.SubscriptionID, cs.ctx.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var stacks []api.Stack
	for _, group := range containerGroups {
		if _, found := group.Tags[composeContainerTag]; !found {
			continue
		}
		state := api.RUNNING
		for _, container := range *group.ContainerGroupProperties.Containers {
			containerState := convert.GetStatus(container, group)
			if containerState != api.RUNNING {
				state = containerState
				break
			}
		}
		stacks = append(stacks, api.Stack{
			ID:     *group.ID,
			Name:   *group.Name,
			Status: state,
		})
	}
	return stacks, nil
}

func checkUnSupportedListOptions(o api.ListOptions) error {
	var errs *multierror.Error
	errs = utils.CheckUnsupported(errs, o.All, false, "all")
	return errs.ErrorOrNil()
}

func (cs *aciComposeService) Logs(ctx context.Context, projectName string, consumer api.LogConsumer, options api.LogOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Convert(ctx context.Context, project *types.Project, options api.ConvertOptions) ([]byte, error) {
	return nil, api.ErrNotImplemented
}

func (cs *aciComposeService) Kill(ctx context.Context, project *types.Project, options api.KillOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) RunOneOffContainer(ctx context.Context, project *types.Project, opts api.RunOptions) (int, error) {
	return 0, api.ErrNotImplemented
}

func (cs *aciComposeService) Remove(ctx context.Context, project *types.Project, options api.RemoveOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Exec(ctx context.Context, project *types.Project, opts api.RunOptions) (int, error) {
	return 0, api.ErrNotImplemented
}
func (cs *aciComposeService) Top(ctx context.Context, projectName string, services []string) ([]api.ContainerProcSummary, error) {
	return nil, api.ErrNotImplemented
}

func (cs *aciComposeService) Events(ctx context.Context, project string, options api.EventsOptions) error {
	return api.ErrNotImplemented
}

func (cs *aciComposeService) Port(ctx context.Context, project string, service string, port int, options api.PortOptions) (string, int, error) {
	return "", 0, api.ErrNotImplemented
}

func (cs *aciComposeService) Images(ctx context.Context, projectName string, options api.ImagesOptions) ([]api.ImageSummary, error) {
	return nil, api.ErrNotImplemented
}
