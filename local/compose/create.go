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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/types"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	volume_api "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/docker/compose-cli/api/compose"
	convert "github.com/docker/compose-cli/local/moby"
	"github.com/docker/compose-cli/progress"
)

func (s *composeService) Create(ctx context.Context, project *types.Project, opts compose.CreateOptions) error {
	err := s.ensureImagesExists(ctx, project)
	if err != nil {
		return err
	}

	if err := s.ensureProjectNetworks(ctx, project); err != nil {
		return err
	}

	if err := s.ensureProjectVolumes(ctx, project); err != nil {
		return err
	}

	var observedState Containers
	observedState, err = s.apiClient.ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(
			projectFilter(project.Name),
		),
		All: true,
	})
	if err != nil {
		return err
	}

	orphans := observedState.filter(isNotService(project.ServiceNames()...))
	if len(orphans) > 0 {
		if opts.RemoveOrphans {
			eg, _ := errgroup.WithContext(ctx)
			w := progress.ContextWriter(ctx)
			s.removeContainers(ctx, w, eg, orphans)
			if eg.Wait() != nil {
				return err
			}
		} else {
			logrus.Warnf("Found orphan containers (%s) for this project. If "+
				"you removed or renamed this service in your compose "+
				"file, you can run this command with the "+
				"--remove-orphans flag to clean it up.", orphans.names())
		}
	}

	return InDependencyOrder(ctx, project, func(c context.Context, service types.ServiceConfig) error {
		return s.ensureService(c, observedState, project, service)
	})
}

func (s *composeService) ensureProjectNetworks(ctx context.Context, project *types.Project) error {
	for k, network := range project.Networks {
		if !network.External.External && network.Name != "" {
			network.Name = fmt.Sprintf("%s_%s", project.Name, k)
			project.Networks[k] = network
		}
		network.Labels = network.Labels.Add(networkLabel, k)
		network.Labels = network.Labels.Add(projectLabel, project.Name)
		network.Labels = network.Labels.Add(versionLabel, ComposeVersion)
		err := s.ensureNetwork(ctx, network)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *composeService) ensureProjectVolumes(ctx context.Context, project *types.Project) error {
	for k, volume := range project.Volumes {
		if !volume.External.External && volume.Name != "" {
			volume.Name = fmt.Sprintf("%s_%s", project.Name, k)
			project.Volumes[k] = volume
		}
		volume.Labels = volume.Labels.Add(volumeLabel, k)
		volume.Labels = volume.Labels.Add(projectLabel, project.Name)
		volume.Labels = volume.Labels.Add(versionLabel, ComposeVersion)
		err := s.ensureVolume(ctx, volume)
		if err != nil {
			return err
		}
	}
	return nil
}

func getImageName(service types.ServiceConfig, projectName string) string {
	imageName := service.Image
	if imageName == "" {
		imageName = projectName + "_" + service.Name
	}
	return imageName
}

func getCreateOptions(p *types.Project, s types.ServiceConfig, number int, inherit *moby.Container, autoRemove bool) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	hash, err := jsonHash(s)
	if err != nil {
		return nil, nil, nil, err
	}

	labels := map[string]string{}
	for k, v := range s.Labels {
		labels[k] = v
	}

	labels[projectLabel] = p.Name
	labels[serviceLabel] = s.Name
	labels[versionLabel] = ComposeVersion
	if _, ok := s.Labels[oneoffLabel]; !ok {
		labels[oneoffLabel] = "False"
	}
	labels[configHashLabel] = hash
	labels[workingDirLabel] = p.WorkingDir
	labels[configFilesLabel] = strings.Join(p.ComposeFiles, ",")
	labels[containerNumberLabel] = strconv.Itoa(number)

	var (
		runCmd     strslice.StrSlice
		entrypoint strslice.StrSlice
	)
	if len(s.Command) > 0 {
		runCmd = strslice.StrSlice(s.Command)
	}
	if len(s.Entrypoint) > 0 {
		entrypoint = strslice.StrSlice(s.Entrypoint)
	}

	var (
		tty         = s.Tty
		stdinOpen   = s.StdinOpen
		attachStdin = false
	)

	containerConfig := container.Config{
		Hostname:        s.Hostname,
		Domainname:      s.DomainName,
		User:            s.User,
		ExposedPorts:    buildContainerPorts(s),
		Tty:             tty,
		OpenStdin:       stdinOpen,
		StdinOnce:       true,
		AttachStdin:     attachStdin,
		AttachStderr:    true,
		AttachStdout:    true,
		Cmd:             runCmd,
		Image:           getImageName(s, p.Name),
		WorkingDir:      s.WorkingDir,
		Entrypoint:      entrypoint,
		NetworkDisabled: s.NetworkMode == "disabled",
		MacAddress:      s.MacAddress,
		Labels:          labels,
		StopSignal:      s.StopSignal,
		Env:             convert.ToMobyEnv(s.Environment),
		Healthcheck:     convert.ToMobyHealthCheck(s.HealthCheck),
		// Volumes:         // FIXME unclear to me the overlap with HostConfig.Mounts
		StopTimeout: convert.ToSeconds(s.StopGracePeriod),
	}

	mountOptions, err := buildContainerMountOptions(*p, s, inherit)
	if err != nil {
		return nil, nil, nil, err
	}
	bindings := buildContainerBindingOptions(s)

	resources := getDeployResources(s)
	networkMode := getNetworkMode(p, s)
	hostConfig := container.HostConfig{
		AutoRemove:     autoRemove,
		Mounts:         mountOptions,
		CapAdd:         strslice.StrSlice(s.CapAdd),
		CapDrop:        strslice.StrSlice(s.CapDrop),
		NetworkMode:    networkMode,
		Init:           s.Init,
		ReadonlyRootfs: s.ReadOnly,
		// ShmSize: , TODO
		Sysctls:      s.Sysctls,
		PortBindings: bindings,
		Resources:    resources,
	}

	networkConfig := buildDefaultNetworkConfig(s, networkMode, getContainerName(p.Name, s, number))
	return &containerConfig, &hostConfig, networkConfig, nil
}

func getDeployResources(s types.ServiceConfig) container.Resources {
	resources := container.Resources{}
	if s.Deploy == nil {
		return resources
	}

	reservations := s.Deploy.Resources.Reservations

	if reservations == nil || len(reservations.Devices) == 0 {
		return resources
	}

	for _, device := range reservations.Devices {
		resources.DeviceRequests = append(resources.DeviceRequests, container.DeviceRequest{
			Capabilities: [][]string{device.Capabilities},
			Count:        int(device.Count),
			DeviceIDs:    device.IDs,
			Driver:       device.Driver,
		})
	}
	return resources
}

func buildContainerPorts(s types.ServiceConfig) nat.PortSet {
	ports := nat.PortSet{}
	for _, p := range s.Ports {
		p := nat.Port(fmt.Sprintf("%d/%s", p.Target, p.Protocol))
		ports[p] = struct{}{}
	}
	return ports
}

func buildContainerBindingOptions(s types.ServiceConfig) nat.PortMap {
	bindings := nat.PortMap{}
	for _, port := range s.Ports {
		p := nat.Port(fmt.Sprintf("%d/%s", port.Target, port.Protocol))
		bind := []nat.PortBinding{}
		binding := nat.PortBinding{}
		if port.Published > 0 {
			binding.HostPort = fmt.Sprint(port.Published)
		}
		bind = append(bind, binding)
		bindings[p] = bind
	}
	return bindings
}

func buildContainerMountOptions(p types.Project, s types.ServiceConfig, inherit *moby.Container) ([]mount.Mount, error) {
	mounts := []mount.Mount{}
	var inherited []string
	if inherit != nil {
		for _, m := range inherit.Mounts {
			if m.Type == "tmpfs" {
				continue
			}
			src := m.Source
			if m.Type == "volume" {
				src = m.Name
			}
			mounts = append(mounts, mount.Mount{
				Type:     m.Type,
				Source:   src,
				Target:   m.Destination,
				ReadOnly: !m.RW,
			})
			inherited = append(inherited, m.Destination)
		}
	}

	for _, v := range s.Volumes {
		if contains(inherited, v.Target) {
			continue
		}
		mount, err := buildMount(p, v)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}

	secretsDir := "/run/secrets"
	for _, secret := range s.Secrets {
		target := secret.Target
		if secret.Target == "" {
			target = filepath.Join(secretsDir, secret.Source)
		} else if !filepath.IsAbs(secret.Target) {
			target = filepath.Join(secretsDir, secret.Target)
		}

		definedSecret := p.Secrets[secret.Source]
		if definedSecret.External.External {
			return nil, fmt.Errorf("unsupported external secret %s", definedSecret.Name)
		}

		if contains(inherited, target) {
			// remove inherited mount
			pos := indexOf(inherited, target)
			if pos >= 0 {
				mounts = append(mounts[:pos], mounts[pos+1])
				inherited = append(inherited[:pos], inherited[pos+1])
			}
		}

		mount, err := buildMount(p, types.ServiceVolumeConfig{
			Type:   types.VolumeTypeBind,
			Source: definedSecret.File,
			Target: target,
		})
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}

	return mounts, nil
}

func buildMount(project types.Project, volume types.ServiceVolumeConfig) (mount.Mount, error) {
	source := volume.Source
	if volume.Type == types.VolumeTypeBind && !filepath.IsAbs(source) {
		// volume source has already been prefixed with workdir if required, by compose-go project loader
		var err error
		source, err = filepath.Abs(source)
		if err != nil {
			return mount.Mount{}, err
		}
	}
	if volume.Type == types.VolumeTypeVolume {
		pVolume, ok := project.Volumes[volume.Source]
		if ok {
			source = pVolume.Name
		}
	}

	return mount.Mount{
		Type:          mount.Type(volume.Type),
		Source:        source,
		Target:        volume.Target,
		ReadOnly:      volume.ReadOnly,
		Consistency:   mount.Consistency(volume.Consistency),
		BindOptions:   buildBindOption(volume.Bind),
		VolumeOptions: buildVolumeOptions(volume.Volume),
		TmpfsOptions:  buildTmpfsOptions(volume.Tmpfs),
	}, nil
}

func buildBindOption(bind *types.ServiceVolumeBind) *mount.BindOptions {
	if bind == nil {
		return nil
	}
	return &mount.BindOptions{
		Propagation: mount.Propagation(bind.Propagation),
		// NonRecursive: false, FIXME missing from model ?
	}
}

func buildVolumeOptions(vol *types.ServiceVolumeVolume) *mount.VolumeOptions {
	if vol == nil {
		return nil
	}
	return &mount.VolumeOptions{
		NoCopy: vol.NoCopy,
		// Labels:       , // FIXME missing from model ?
		// DriverConfig: , // FIXME missing from model ?
	}
}

func buildTmpfsOptions(tmpfs *types.ServiceVolumeTmpfs) *mount.TmpfsOptions {
	if tmpfs == nil {
		return nil
	}
	return &mount.TmpfsOptions{
		SizeBytes: tmpfs.Size,
		// Mode:      , // FIXME missing from model ?
	}
}

func buildDefaultNetworkConfig(s types.ServiceConfig, networkMode container.NetworkMode, containerName string) *network.NetworkingConfig {
	config := map[string]*network.EndpointSettings{}
	net := string(networkMode)
	config[net] = &network.EndpointSettings{
		Aliases: append(getAliases(s, s.Networks[net]), containerName),
	}

	return &network.NetworkingConfig{
		EndpointsConfig: config,
	}
}

func getAliases(s types.ServiceConfig, c *types.ServiceNetworkConfig) []string {
	aliases := []string{s.Name}
	if c != nil {
		aliases = append(aliases, c.Aliases...)
	}
	return aliases
}

func getNetworkMode(p *types.Project, service types.ServiceConfig) container.NetworkMode {
	mode := service.NetworkMode
	if mode == "" {
		if len(p.Networks) > 0 {
			for name := range getNetworksForService(service) {
				return container.NetworkMode(p.Networks[name].Name)
			}
		}
		return container.NetworkMode("none")
	}

	// FIXME incomplete implementation
	if strings.HasPrefix(mode, "service:") {
		panic("Not yet implemented")
	}
	if strings.HasPrefix(mode, "container:") {
		panic("Not yet implemented")
	}

	return container.NetworkMode(mode)
}

func getNetworksForService(s types.ServiceConfig) map[string]*types.ServiceNetworkConfig {
	if len(s.Networks) > 0 {
		return s.Networks
	}
	return map[string]*types.ServiceNetworkConfig{"default": nil}
}

func (s *composeService) ensureNetwork(ctx context.Context, n types.NetworkConfig) error {
	_, err := s.apiClient.NetworkInspect(ctx, n.Name, moby.NetworkInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			if n.External.External {
				return fmt.Errorf("network %s declared as external, but could not be found", n.Name)
			}
			createOpts := moby.NetworkCreate{
				// TODO NameSpace Labels
				Labels:     n.Labels,
				Driver:     n.Driver,
				Options:    n.DriverOpts,
				Internal:   n.Internal,
				Attachable: n.Attachable,
			}

			if n.Ipam.Driver != "" || len(n.Ipam.Config) > 0 {
				createOpts.IPAM = &network.IPAM{}
			}

			if n.Ipam.Driver != "" {
				createOpts.IPAM.Driver = n.Ipam.Driver
			}

			for _, ipamConfig := range n.Ipam.Config {
				config := network.IPAMConfig{
					Subnet: ipamConfig.Subnet,
				}
				createOpts.IPAM.Config = append(createOpts.IPAM.Config, config)
			}
			networkEventName := fmt.Sprintf("Network %q", n.Name)
			w := progress.ContextWriter(ctx)
			w.Event(progress.CreatingEvent(networkEventName))
			if _, err := s.apiClient.NetworkCreate(ctx, n.Name, createOpts); err != nil {
				w.Event(progress.ErrorEvent(networkEventName))
				return errors.Wrapf(err, "failed to create network %s", n.Name)
			}
			w.Event(progress.CreatedEvent(networkEventName))
			return nil
		}
		return err
	}
	return nil
}

func (s *composeService) ensureNetworkDown(ctx context.Context, networkID string, networkName string) error {
	w := progress.ContextWriter(ctx)
	eventName := fmt.Sprintf("Network %q", networkName)
	w.Event(progress.RemovingEvent(eventName))

	if err := s.apiClient.NetworkRemove(ctx, networkID); err != nil {
		w.Event(progress.ErrorEvent(eventName))
		return errors.Wrapf(err, fmt.Sprintf("failed to create network %s", networkID))
	}

	w.Event(progress.RemovedEvent(eventName))
	return nil
}

func (s *composeService) ensureVolume(ctx context.Context, volume types.VolumeConfig) error {
	// TODO could identify volume by label vs name
	_, err := s.apiClient.VolumeInspect(ctx, volume.Name)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return err
		}
		eventName := fmt.Sprintf("Volume %q", volume.Name)
		w := progress.ContextWriter(ctx)
		w.Event(progress.CreatingEvent(eventName))
		// TODO we miss support for driver_opts and labels
		_, err := s.apiClient.VolumeCreate(ctx, volume_api.VolumeCreateBody{
			Labels:     volume.Labels,
			Name:       volume.Name,
			Driver:     volume.Driver,
			DriverOpts: volume.DriverOpts,
		})
		if err != nil {
			w.Event(progress.ErrorEvent(eventName))
			return err
		}
		w.Event(progress.CreatedEvent(eventName))
	}
	return nil
}
