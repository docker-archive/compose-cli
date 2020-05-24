package azure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/docker/api/context/cloud"
	"github.com/docker/api/errdefs"
	"github.com/docker/api/progress"

	"github.com/Azure/azure-sdk-for-go/services/containerinstance/mgmt/2018-10-01/containerinstance"
	"github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/docker/api/azure/convert"
	"github.com/docker/api/azure/login"
	"github.com/docker/api/backend"
	"github.com/docker/api/compose"
	"github.com/docker/api/containers"
	apicontext "github.com/docker/api/context"
	"github.com/docker/api/context/store"
)

const singleContainerName = "single--container--aci"

// ErrNoSuchContainer is returned when the mentioned container does not exist
var ErrNoSuchContainer = errors.New("no such container")

func init() {
	backend.Register("aci", "aci", func(ctx context.Context) (backend.Service, error) {
		return New(ctx)
	})
}

func getter() interface{} {
	return &store.AciContext{}
}

// New creates a backend that can manage containers
func New(ctx context.Context) (backend.Service, error) {
	currentContext := apicontext.CurrentContext(ctx)
	contextStore := store.ContextStore(ctx)
	metadata, err := contextStore.Get(currentContext, getter)
	if err != nil {
		return nil, errors.Wrap(err, "wrong context type")
	}
	aciContext, _ := metadata.Metadata.Data.(store.AciContext)

	auth, _ := login.NewAuthorizerFromLogin()
	containerGroupsClient := containerinstance.NewContainerGroupsClient(aciContext.SubscriptionID)
	containerGroupsClient.Authorizer = auth

	return getAciAPIService(containerGroupsClient, aciContext)
}

func getAciAPIService(cgc containerinstance.ContainerGroupsClient, aciCtx store.AciContext) (*aciAPIService, error) {
	service, err := login.NewAzureLoginService()
	if err != nil {
		return nil, err
	}
	return &aciAPIService{
		aciContainerService: aciContainerService{
			containerGroupsClient: cgc,
			ctx:                   aciCtx,
		},
		aciComposeService: aciComposeService{
			ctx: aciCtx,
		},
		aciCloudService: aciCloudService{
			loginService: service,
		},
	}, nil
}

type aciAPIService struct {
	aciContainerService
	aciComposeService
	aciCloudService
}

func (a *aciAPIService) ContainerService() containers.Service {
	return &a.aciContainerService
}

func (a *aciAPIService) ComposeService() compose.Service {
	return &a.aciComposeService
}

func (a *aciAPIService) CloudService() cloud.Service {
	return &a.aciCloudService
}

type aciContainerService struct {
	containerGroupsClient containerinstance.ContainerGroupsClient
	ctx                   store.AciContext
}

func (cs *aciContainerService) List(ctx context.Context, _ bool) ([]containers.Container, error) {
	var containerGroups []containerinstance.ContainerGroup
	result, err := cs.containerGroupsClient.ListByResourceGroup(ctx, cs.ctx.ResourceGroup)
	if err != nil {
		return []containers.Container{}, err
	}

	for result.NotDone() {
		containerGroups = append(containerGroups, result.Values()...)
		if err := result.NextWithContext(ctx); err != nil {
			return []containers.Container{}, err
		}
	}

	var res []containers.Container
	for _, containerGroup := range containerGroups {
		group, err := cs.containerGroupsClient.Get(ctx, cs.ctx.ResourceGroup, *containerGroup.Name)
		if err != nil {
			return []containers.Container{}, err
		}

		for _, container := range *group.Containers {
			var containerID string
			if *container.Name == singleContainerName {
				containerID = *containerGroup.Name
			} else {
				containerID = *containerGroup.Name + "_" + *container.Name
			}
			status := "Unknown"
			if container.InstanceView != nil && container.InstanceView.CurrentState != nil {
				status = *container.InstanceView.CurrentState.State
			}

			res = append(res, containers.Container{
				ID:     containerID,
				Image:  *container.Image,
				Status: status,
				Ports:  convert.ToPorts(group.IPAddress, *container.Ports),
			})
		}
	}

	return res, nil
}

func (cs *aciContainerService) Run(ctx context.Context, c chan<- progress.Event, r containers.ContainerConfig) error {
	var ports []types.ServicePortConfig
	for _, p := range r.Ports {
		ports = append(ports, types.ServicePortConfig{
			Target:    p.ContainerPort,
			Published: p.HostPort,
		})
	}

	projectVolumes, serviceConfigVolumes, err := convert.GetRunVolumes(r.Volumes)
	if err != nil {
		return err
	}

	project := compose.Project{
		Name: r.ID,
		Config: types.Config{
			Services: []types.ServiceConfig{
				{
					Name:    singleContainerName,
					Image:   r.Image,
					Ports:   ports,
					Labels:  r.Labels,
					Volumes: serviceConfigVolumes,
				},
			},
			Volumes: projectVolumes,
		},
	}

	logrus.Debugf("Running container %q with name %q\n", r.Image, r.ID)
	groupDefinition, err := convert.ToContainerGroup(cs.ctx, project)
	if err != nil {
		return err
	}

	return createACIContainers(ctx, cs.ctx, c, groupDefinition)
}

func (cs *aciContainerService) Stop(ctx context.Context, containerName string, timeout *uint32) error {
	return errdefs.ErrNotImplemented
}

func getGroupAndContainerName(containerID string) (groupName string, containerName string) {
	tokens := strings.Split(containerID, "_")
	groupName = tokens[0]
	if len(tokens) > 1 {
		containerName = tokens[len(tokens)-1]
		groupName = containerID[:len(containerID)-(len(containerName)+1)]
	} else {
		containerName = singleContainerName
	}
	return groupName, containerName
}

func (cs *aciContainerService) Exec(ctx context.Context, name string, command string, reader io.Reader, writer io.Writer) error {
	groupName, containerAciName := getGroupAndContainerName(name)
	containerExecResponse, err := execACIContainer(ctx, cs.ctx, command, groupName, containerAciName)
	if err != nil {
		return err
	}

	return exec(
		context.Background(),
		*containerExecResponse.WebSocketURI,
		*containerExecResponse.Password,
		reader,
		writer,
	)
}

func (cs *aciContainerService) Logs(ctx context.Context, containerName string, req containers.LogsRequest) error {
	groupName, containerAciName := getGroupAndContainerName(containerName)
	logs, err := getACIContainerLogs(ctx, cs.ctx, groupName, containerAciName)
	if err != nil {
		return err
	}
	if req.Tail != "all" {
		tail, err := strconv.Atoi(req.Tail)
		if err != nil {
			return err
		}
		lines := strings.Split(logs, "\n")

		// If asked for less lines than exist, take only those lines
		if tail <= len(lines) {
			logs = strings.Join(lines[len(lines)-tail:], "\n")
		}
	}

	_, err = fmt.Fprint(req.Writer, logs)
	return err
}

func (cs *aciContainerService) Delete(ctx context.Context, containerID string, _ bool) error {
	cg, err := deleteACIContainerGroup(ctx, cs.ctx, containerID)
	if err != nil {
		return err
	}
	if cg.StatusCode == http.StatusNoContent {
		return ErrNoSuchContainer
	}

	return err
}

type aciComposeService struct {
	ctx store.AciContext
}

func (cs *aciComposeService) Up(ctx context.Context, opts compose.ProjectOptions, ch chan<- progress.Event) error {
	project, err := compose.ProjectFromOptions(&opts)
	if err != nil {
		return err
	}
	logrus.Debugf("Up on project with name %q\n", project.Name)
	groupDefinition, err := convert.ToContainerGroup(cs.ctx, *project)

	if err != nil {
		return err
	}
	return createACIContainers(ctx, cs.ctx, ch, groupDefinition)
}

func (cs *aciComposeService) Down(ctx context.Context, opts compose.ProjectOptions) error {
	project, err := compose.ProjectFromOptions(&opts)
	if err != nil {
		return err
	}
	logrus.Debugf("Down on project with name %q\n", project.Name)

	cg, err := deleteACIContainerGroup(ctx, cs.ctx, project.Name)
	if err != nil {
		return err
	}
	if cg.StatusCode == http.StatusNoContent {
		return ErrNoSuchContainer
	}

	return err
}

type aciCloudService struct {
	loginService login.AzureLoginService
}

func (cs *aciCloudService) Login(ctx context.Context, params map[string]string) error {
	return cs.loginService.Login(ctx)
}
