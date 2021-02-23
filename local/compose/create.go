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

	"github.com/docker/compose-cli/api/compose"
	"github.com/docker/compose-cli/api/progress"
	convert "github.com/docker/compose-cli/local/moby"
)

func (s *composeService) Create(ctx context.Context, project *types.Project, opts compose.CreateOptions) error {
	err := s.ensureImagesExists(ctx, project)
	if err != nil {
		return err
	}

	prepareNetworks(project)

	err = prepareVolumes(project)
	if err != nil {
		return err
	}

	if err := s.ensureNetworks(ctx, project.Networks); err != nil {
		return err
	}

	if err := s.ensureProjectVolumes(ctx, project); err != nil {
		return err
	}

	var observedState Containers
	observedState, err = s.apiClient.ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(projectFilter(project.Name)),
		All:     true,
	})
	if err != nil {
		return err
	}
	containerState := NewContainersState(observedState)
	ctx = context.WithValue(ctx, ContainersKey{}, containerState)

	allServices := project.AllServices()
	allServiceNames := []string{}
	for _, service := range allServices {
		allServiceNames = append(allServiceNames, service.Name)
	}
	orphans := observedState.filter(isNotService(allServiceNames...))
	if len(orphans) > 0 {
		if opts.RemoveOrphans {
			w := progress.ContextWriter(ctx)
			err := s.removeContainers(ctx, w, orphans, nil)
			if err != nil {
				return err
			}
		} else {
			logrus.Warnf("Found orphan containers (%s) for this project. If "+
				"you removed or renamed this service in your compose "+
				"file, you can run this command with the "+
				"--remove-orphans flag to clean it up.", orphans.names())
		}
	}

	prepareNetworkMode(project)

	return InDependencyOrder(ctx, project, func(c context.Context, service types.ServiceConfig) error {
		return s.ensureService(c, project, service, opts.Recreate)
	})
}

func prepareVolumes(p *types.Project) error {
	for i := range p.Services {
		volumesFrom, dependServices, err := getVolumesFrom(p, p.Services[i].VolumesFrom)
		if err != nil {
			return err
		}
		p.Services[i].VolumesFrom = volumesFrom
		if len(dependServices) > 0 {
			if p.Services[i].DependsOn == nil {
				p.Services[i].DependsOn = make(types.DependsOnConfig, len(dependServices))
			}
			for _, service := range p.Services {
				if contains(dependServices, service.Name) {
					p.Services[i].DependsOn[service.Name] = types.ServiceDependency{
						Condition: types.ServiceConditionStarted,
					}
				}
			}
		}
	}
	return nil
}

func prepareNetworks(project *types.Project) {
	for k, network := range project.Networks {
		network.Labels = network.Labels.Add(networkLabel, k)
		network.Labels = network.Labels.Add(projectLabel, project.Name)
		network.Labels = network.Labels.Add(versionLabel, ComposeVersion)
		project.Networks[k] = network
	}
}

func prepareNetworkMode(p *types.Project) {
outLoop:
	for i := range p.Services {
		dependency := getDependentServiceByNetwork(p.Services[i].NetworkMode)
		if dependency == "" {
			continue
		}
		if p.Services[i].DependsOn == nil {
			p.Services[i].DependsOn = make(types.DependsOnConfig)
		}
		for _, service := range p.Services {
			if service.Name == dependency {
				p.Services[i].DependsOn[service.Name] = types.ServiceDependency{
					Condition: types.ServiceConditionStarted,
				}
				continue outLoop
			}
		}
	}
}

func (s *composeService) ensureNetworks(ctx context.Context, networks types.Networks) error {
	for _, network := range networks {
		err := s.ensureNetwork(ctx, network)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *composeService) ensureProjectVolumes(ctx context.Context, project *types.Project) error {
	for k, volume := range project.Volumes {
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

func (s *composeService) getCreateOptions(ctx context.Context, p *types.Project, service types.ServiceConfig, number int, inherit *moby.Container,
	autoRemove bool) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {

	hash, err := jsonHash(service)
	if err != nil {
		return nil, nil, nil, err
	}

	labels := map[string]string{}
	for k, v := range service.Labels {
		labels[k] = v
	}

	labels[projectLabel] = p.Name
	labels[serviceLabel] = service.Name
	labels[versionLabel] = ComposeVersion
	if _, ok := service.Labels[oneoffLabel]; !ok {
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
	if len(service.Command) > 0 {
		runCmd = strslice.StrSlice(service.Command)
	}
	if len(service.Entrypoint) > 0 {
		entrypoint = strslice.StrSlice(service.Entrypoint)
	}

	var (
		tty         = service.Tty
		stdinOpen   = service.StdinOpen
		attachStdin = false
	)

	volumeMounts, binds, mounts, err := s.buildContainerVolumes(ctx, *p, service, inherit)
	if err != nil {
		return nil, nil, nil, err
	}

	containerConfig := container.Config{
		Hostname:        service.Hostname,
		Domainname:      service.DomainName,
		User:            service.User,
		ExposedPorts:    buildContainerPorts(service),
		Tty:             tty,
		OpenStdin:       stdinOpen,
		StdinOnce:       true,
		AttachStdin:     attachStdin,
		AttachStderr:    true,
		AttachStdout:    true,
		Cmd:             runCmd,
		Image:           getImageName(service, p.Name),
		WorkingDir:      service.WorkingDir,
		Entrypoint:      entrypoint,
		NetworkDisabled: service.NetworkMode == "disabled",
		MacAddress:      service.MacAddress,
		Labels:          labels,
		StopSignal:      service.StopSignal,
		Env:             convert.ToMobyEnv(service.Environment),
		Healthcheck:     convert.ToMobyHealthCheck(service.HealthCheck),
		Volumes:         volumeMounts,

		StopTimeout: convert.ToSeconds(service.StopGracePeriod),
	}

	portBindings := buildContainerPortBindingOptions(service)

	resources := getDeployResources(service)

	networkMode, err := getNetworkMode(ctx, p, service)
	if err != nil {
		return nil, nil, nil, err
	}
	hostConfig := container.HostConfig{
		AutoRemove:     autoRemove,
		Binds:          binds,
		Mounts:         mounts,
		CapAdd:         strslice.StrSlice(service.CapAdd),
		CapDrop:        strslice.StrSlice(service.CapDrop),
		NetworkMode:    networkMode,
		Init:           service.Init,
		ReadonlyRootfs: service.ReadOnly,
		// ShmSize: , TODO
		Sysctls:      service.Sysctls,
		PortBindings: portBindings,
		Resources:    resources,
		VolumeDriver: service.VolumeDriver,
		VolumesFrom:  service.VolumesFrom,
	}

	networkConfig := buildDefaultNetworkConfig(service, networkMode, getContainerName(p.Name, service, number))
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

func buildContainerPortBindingOptions(s types.ServiceConfig) nat.PortMap {
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

func getVolumesFrom(project *types.Project, volumesFrom []string) ([]string, []string, error) {
	var volumes = []string{}
	var services = []string{}
	// parse volumes_from
	if len(volumesFrom) == 0 {
		return volumes, services, nil
	}
	for _, vol := range volumesFrom {
		spec := strings.Split(vol, ":")
		if len(spec) == 0 {
			continue
		}
		if spec[0] == "container" {
			volumes = append(volumes, strings.Join(spec[1:], ":"))
			continue
		}
		serviceName := spec[0]
		services = append(services, serviceName)
		service, err := project.GetService(serviceName)
		if err != nil {
			return nil, nil, err
		}

		firstContainer := getContainerName(project.Name, service, 1)
		v := fmt.Sprintf("%s:%s", firstContainer, strings.Join(spec[1:], ":"))
		volumes = append(volumes, v)
	}
	return volumes, services, nil

}

func getDependentServiceByNetwork(networkMode string) string {
	baseService := ""
	if strings.HasPrefix(networkMode, types.NetworkModeServicePrefix) {
		return networkMode[len(types.NetworkModeServicePrefix):]
	}
	return baseService
}

func (s *composeService) buildContainerVolumes(ctx context.Context, p types.Project, service types.ServiceConfig,
	inherit *moby.Container) (map[string]struct{}, []string, []mount.Mount, error) {
	var mounts = []mount.Mount{}

	image := getImageName(service, p.Name)
	imgInspect, _, err := s.apiClient.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return nil, nil, nil, err
	}

	mountOptions, err := buildContainerMountOptions(p, service, imgInspect, inherit)
	if err != nil {
		return nil, nil, nil, err
	}

	// filter binds and volumes mount targets
	volumeMounts := map[string]struct{}{}
	binds := []string{}
	for _, m := range mountOptions {

		if m.Type == mount.TypeVolume {
			volumeMounts[m.Target] = struct{}{}
			if m.Source != "" {
				binds = append(binds, fmt.Sprintf("%s:%s:rw", m.Source, m.Target))
			}
		}
	}
	for _, m := range mountOptions {
		if m.Type == mount.TypeBind || m.Type == mount.TypeTmpfs {
			mounts = append(mounts, m)
		}
	}
	return volumeMounts, binds, mounts, nil
}

func buildContainerMountOptions(p types.Project, s types.ServiceConfig, img moby.ImageInspect, inherit *moby.Container) ([]mount.Mount, error) {
	var mounts = map[string]mount.Mount{}
	if inherit != nil {
		for _, m := range inherit.Mounts {
			if m.Type == "tmpfs" {
				continue
			}
			src := m.Source
			if m.Type == "volume" {
				src = m.Name
			}
			mounts[m.Destination] = mount.Mount{
				Type:     m.Type,
				Source:   src,
				Target:   m.Destination,
				ReadOnly: !m.RW,
			}
		}
	}
	if img.ContainerConfig != nil {
		for k := range img.ContainerConfig.Volumes {
			m, err := buildMount(p, types.ServiceVolumeConfig{
				Type:   types.VolumeTypeVolume,
				Target: k,
			})
			if err != nil {
				return nil, err
			}
			mounts[k] = m

		}
	}

	mounts, err := fillBindMounts(p, s, mounts)
	if err != nil {
		return nil, err
	}

	values := make([]mount.Mount, 0, len(mounts))
	for _, v := range mounts {
		values = append(values, v)
	}
	return values, nil
}

func fillBindMounts(p types.Project, s types.ServiceConfig, m map[string]mount.Mount) (map[string]mount.Mount, error) {
	for _, v := range s.Volumes {
		bindMount, err := buildMount(p, v)
		if err != nil {
			return nil, err
		}
		m[bindMount.Target] = bindMount
	}

	secrets, err := buildContainerSecretMounts(p, s)
	if err != nil {
		return nil, err
	}
	for _, s := range secrets {
		if _, found := m[s.Target]; found {
			continue
		}
		m[s.Target] = s
	}

	configs, err := buildContainerConfigMounts(p, s)
	if err != nil {
		return nil, err
	}
	for _, c := range configs {
		if _, found := m[c.Target]; found {
			continue
		}
		m[c.Target] = c
	}
	return m, nil
}

func buildContainerConfigMounts(p types.Project, s types.ServiceConfig) ([]mount.Mount, error) {
	var mounts = map[string]mount.Mount{}

	configsBaseDir := "/"
	for _, config := range s.Configs {
		target := config.Target
		if config.Target == "" {
			target = filepath.Join(configsBaseDir, config.Source)
		} else if !filepath.IsAbs(config.Target) {
			target = filepath.Join(configsBaseDir, config.Target)
		}

		definedConfig := p.Configs[config.Source]
		if definedConfig.External.External {
			return nil, fmt.Errorf("unsupported external config %s", definedConfig.Name)
		}

		bindMount, err := buildMount(p, types.ServiceVolumeConfig{
			Type:     types.VolumeTypeBind,
			Source:   definedConfig.File,
			Target:   target,
			ReadOnly: true,
		})
		if err != nil {
			return nil, err
		}
		mounts[target] = bindMount
	}
	values := make([]mount.Mount, 0, len(mounts))
	for _, v := range mounts {
		values = append(values, v)
	}
	return values, nil
}

func buildContainerSecretMounts(p types.Project, s types.ServiceConfig) ([]mount.Mount, error) {
	var mounts = map[string]mount.Mount{}

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

		mount, err := buildMount(p, types.ServiceVolumeConfig{
			Type:     types.VolumeTypeBind,
			Source:   definedSecret.File,
			Target:   target,
			ReadOnly: true,
		})
		if err != nil {
			return nil, err
		}
		mounts[target] = mount
	}
	values := make([]mount.Mount, 0, len(mounts))
	for _, v := range mounts {
		values = append(values, v)
	}
	return values, nil
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
		if volume.Source != "" {

			pVolume, ok := project.Volumes[volume.Source]
			if ok {
				source = pVolume.Name
			}
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

func getNetworkMode(ctx context.Context, p *types.Project, service types.ServiceConfig) (container.NetworkMode, error) {
	cState, err := GetContextContainerState(ctx)
	if err != nil {
		return container.NetworkMode("none"), nil
	}
	observedState := cState.GetContainers()

	mode := service.NetworkMode
	if mode == "" {
		if len(p.Networks) > 0 {
			for name := range getNetworksForService(service) {
				return container.NetworkMode(p.Networks[name].Name), nil
			}
		}
		return container.NetworkMode("none"), nil
	}
	depServiceNetworkMode := getDependentServiceByNetwork(service.NetworkMode)
	if depServiceNetworkMode != "" {
		depServiceContainers := observedState.filter(isService(depServiceNetworkMode))
		if len(depServiceContainers) > 0 {
			return container.NetworkMode(types.NetworkModeContainerPrefix + depServiceContainers[0].ID), nil
		}
		return container.NetworkMode("none"),
			fmt.Errorf(`no containers started for network_mode %q in service %q -> %v`,
				mode, service.Name, observedState)
	}
	return container.NetworkMode(mode), nil
}

func getNetworksForService(s types.ServiceConfig) map[string]*types.ServiceNetworkConfig {
	if len(s.Networks) > 0 {
		return s.Networks
	}
	if s.NetworkMode != "" {
		return nil
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
