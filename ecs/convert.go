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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/compose-cli/ecs/secrets"

	ecsapi "github.com/aws/aws-sdk-go/service/ecs"
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/opts"
	"github.com/joho/godotenv"
)

const secretsInitContainerImage = "docker/ecs-secrets-sidecar:1.0"
const searchDomainInitContainerImage = "docker/ecs-searchdomain-sidecar:1.0"

func (b *ecsAPIService) createTaskDefinition(project *types.Project, service types.ServiceConfig, resources awsResources) (*ecs.TaskDefinition, error) {
	cpu, mem, err := toLimits(service)
	if err != nil {
		return nil, err
	}
	_, memReservation := toContainerReservation(service)
	credential := getRepoCredentials(service)

	logConfiguration := getLogConfiguration(service, project)

	var (
		initContainers []ecs.TaskDefinition_ContainerDefinition
		volumes        []ecs.TaskDefinition_Volume
		mounts         []ecs.TaskDefinition_MountPoint
	)
	if len(service.Secrets) > 0 {
		secretsVolume, secretsMount, secretsSideCar, err := createSecretsSideCar(project, service, logConfiguration)
		if err != nil {
			return nil, err
		}
		initContainers = append(initContainers, secretsSideCar)
		volumes = append(volumes, secretsVolume)
		mounts = append(mounts, secretsMount)
	}

	initContainers = append(initContainers, ecs.TaskDefinition_ContainerDefinition{
		Name:             fmt.Sprintf("%s_ResolvConf_InitContainer", normalizeResourceName(service.Name)),
		Image:            searchDomainInitContainerImage,
		Essential:        cloudformation.Bool(false),
		Command:          []string{b.Region + ".compute.internal", project.Name + ".local"},
		LogConfiguration: logConfiguration,
	})

	var dependencies []ecs.TaskDefinition_ContainerDependency
	for _, c := range initContainers {
		dependencies = append(dependencies, ecs.TaskDefinition_ContainerDependency{
			Condition:     cloudformation.String(ecsapi.ContainerConditionSuccess),
			ContainerName: cloudformation.String(c.Name),
		})
	}

	for _, v := range service.Volumes {
		n := fmt.Sprintf("%sAccessPoint", normalizeResourceName(v.Source))
		volumes = append(volumes, ecs.TaskDefinition_Volume{
			EFSVolumeConfiguration: &ecs.TaskDefinition_EFSVolumeConfiguration{
				AuthorizationConfig: &ecs.TaskDefinition_AuthorizationConfig{
					AccessPointId: cloudformation.RefPtr(n),
					IAM:           cloudformation.String("ENABLED"),
				},
				FilesystemId:      resources.filesystems[v.Source].ID(),
				TransitEncryption: cloudformation.String("ENABLED"),
			},
			Name: cloudformation.String(v.Source),
		})
		mounts = append(mounts, ecs.TaskDefinition_MountPoint{
			ContainerPath: cloudformation.String(v.Target),
			ReadOnly:      cloudformation.Bool(v.ReadOnly),
			SourceVolume:  cloudformation.String(v.Source),
		})
	}

	pairs, err := createEnvironment(project, service)
	if err != nil {
		return nil, err
	}
	var reservations *types.Resource
	if service.Deploy != nil && service.Deploy.Resources.Reservations != nil {
		reservations = service.Deploy.Resources.Reservations
	}

	containers := append(initContainers, ecs.TaskDefinition_ContainerDefinition{
		Command:                service.Command,
		DisableNetworking:      cloudformation.Bool(service.NetworkMode == "none"),
		DependsOnProp:          dependencies,
		DnsSearchDomains:       service.DNSSearch,
		DnsServers:             service.DNS,
		DockerLabels:           service.Labels,
		DockerSecurityOptions:  service.SecurityOpt,
		EntryPoint:             service.Entrypoint,
		Environment:            pairs,
		Essential:              cloudformation.Bool(true),
		ExtraHosts:             toHostEntryPtr(service.ExtraHosts),
		FirelensConfiguration:  nil,
		HealthCheck:            toHealthCheck(service.HealthCheck),
		Hostname:               cloudformation.String(service.Hostname),
		Image:                  service.Image,
		Interactive:            cloudformation.Bool(false),
		Links:                  nil,
		LinuxParameters:        toLinuxParameters(service),
		LogConfiguration:       logConfiguration,
		MemoryReservation:      cloudformation.Int(memReservation),
		MountPoints:            mounts,
		Name:                   service.Name,
		PortMappings:           toPortMappings(service.Ports),
		Privileged:             cloudformation.Bool(service.Privileged),
		PseudoTerminal:         cloudformation.Bool(service.Tty),
		ReadonlyRootFilesystem: cloudformation.Bool(service.ReadOnly),
		RepositoryCredentials:  credential,
		ResourceRequirements:   toTaskResourceRequirements(reservations),
		StartTimeout:           cloudformation.Int(0),
		StopTimeout:            cloudformation.Int(durationToInt(service.StopGracePeriod)),
		SystemControls:         toSystemControls(service.Sysctls),
		Ulimits:                toUlimits(service.Ulimits),
		User:                   cloudformation.String(service.User),
		VolumesFrom:            nil,
		WorkingDirectory:       cloudformation.String(service.WorkingDir),
	})

	launchType := ecsapi.LaunchTypeFargate
	if requireEC2(service) {
		launchType = ecsapi.LaunchTypeEc2
	}

	return &ecs.TaskDefinition{
		ContainerDefinitions: containers,
		Cpu:                  cloudformation.String(cpu),
		Family:               cloudformation.String(fmt.Sprintf("%s-%s", project.Name, service.Name)),
		IpcMode:              cloudformation.String(service.Ipc),
		Memory:               cloudformation.String(mem),
		NetworkMode:          cloudformation.String(ecsapi.NetworkModeAwsvpc), // FIXME could be set by service.NetworkMode, Fargate only supports network mode ‘awsvpc’.
		PidMode:              cloudformation.String(service.Pid),
		PlacementConstraints: toPlacementConstraints(service.Deploy),
		ProxyConfiguration:   nil,
		RequiresCompatibilities: []string{
			launchType,
		},
		Volumes: volumes,
	}, nil
}

func toTaskResourceRequirements(reservations *types.Resource) []ecs.TaskDefinition_ResourceRequirement {
	if reservations == nil {
		return nil
	}
	var requirements []ecs.TaskDefinition_ResourceRequirement
	for _, r := range reservations.GenericResources {
		if r.DiscreteResourceSpec.Kind == "gpus" {
			requirements = append(requirements, ecs.TaskDefinition_ResourceRequirement{
				Type:  ecsapi.ResourceTypeGpu,
				Value: fmt.Sprint(r.DiscreteResourceSpec.Value),
			})
		}
	}
	for _, r := range reservations.Devices {
		hasGpuCap := false
		for _, c := range r.Capabilities {
			if c == "gpu" {
				hasGpuCap = true
				break
			}
		}
		if hasGpuCap {
			count := r.Count
			if count <= 0 {
				count = 1
			}
			requirements = append(requirements, ecs.TaskDefinition_ResourceRequirement{
				Type:  ecsapi.ResourceTypeGpu,
				Value: fmt.Sprint(count),
			})
		}
	}
	return requirements
}

func createSecretsSideCar(project *types.Project, service types.ServiceConfig, logConfiguration *ecs.TaskDefinition_LogConfiguration) (
	ecs.TaskDefinition_Volume,
	ecs.TaskDefinition_MountPoint,
	ecs.TaskDefinition_ContainerDefinition,
	error) {
	initContainerName := fmt.Sprintf("%s_Secrets_InitContainer", normalizeResourceName(service.Name))
	secretsVolume := ecs.TaskDefinition_Volume{
		Name: cloudformation.String("secrets"),
	}
	secretsMount := ecs.TaskDefinition_MountPoint{
		ContainerPath: cloudformation.String("/run/secrets/"),
		ReadOnly:      cloudformation.Bool(true),
		SourceVolume:  cloudformation.String("secrets"),
	}

	var (
		args        []secrets.Secret
		taskSecrets []ecs.TaskDefinition_Secret
	)
	for _, s := range service.Secrets {
		secretConfig := project.Secrets[s.Source]
		if s.Target == "" {
			s.Target = s.Source
		}
		taskSecrets = append(taskSecrets, ecs.TaskDefinition_Secret{
			Name:      s.Target,
			ValueFrom: secretConfig.Name,
		})
		var keys []string
		if ext, ok := secretConfig.Extensions[extensionKeys]; ok {
			if key, ok := ext.(string); ok {
				keys = append(keys, key)
			} else {
				for _, k := range ext.([]interface{}) {
					keys = append(keys, k.(string))
				}
			}
		}
		args = append(args, secrets.Secret{
			Name: s.Target,
			Keys: keys,
		})
	}
	command, err := json.Marshal(args)
	if err != nil {
		return ecs.TaskDefinition_Volume{}, ecs.TaskDefinition_MountPoint{}, ecs.TaskDefinition_ContainerDefinition{}, err
	}
	secretsSideCar := ecs.TaskDefinition_ContainerDefinition{
		Name:             initContainerName,
		Image:            secretsInitContainerImage,
		Command:          []string{string(command)},
		Essential:        cloudformation.Bool(false),
		LogConfiguration: logConfiguration,
		MountPoints: []ecs.TaskDefinition_MountPoint{
			{
				ContainerPath: cloudformation.String("/run/secrets/"),
				ReadOnly:      cloudformation.Bool(false),
				SourceVolume:  cloudformation.String("secrets"),
			},
		},
		Secrets: taskSecrets,
	}
	return secretsVolume, secretsMount, secretsSideCar, nil
}

func createEnvironment(project *types.Project, service types.ServiceConfig) ([]ecs.TaskDefinition_KeyValuePair, error) {
	environment := map[string]*string{}
	for _, f := range service.EnvFile {
		if !filepath.IsAbs(f) {
			f = filepath.Join(project.WorkingDir, f)
		}
		if _, err := os.Stat(f); os.IsNotExist(err) {
			return nil, err
		}
		file, err := os.Open(f)
		if err != nil {
			return nil, err
		}
		defer file.Close() // nolint:errcheck

		env, err := godotenv.Parse(file)
		if err != nil {
			return nil, err
		}
		for k, v := range env {
			environment[k] = &v
		}
	}
	for k, v := range service.Environment {
		environment[k] = v
	}

	var pairs []ecs.TaskDefinition_KeyValuePair
	for k, v := range environment {
		name := k
		var value string
		if v != nil {
			value = *v
		}
		pairs = append(pairs, ecs.TaskDefinition_KeyValuePair{
			Name:  cloudformation.String(name),
			Value: cloudformation.String(value),
		})
	}

	//order env keys for idempotence between calls
	//to avoid unnecessary resource recreations on CloudFormation
	sort.Slice(pairs, func(i, j int) bool {
		return cloudformation.StringValue(pairs[i].Name) < cloudformation.StringValue(pairs[j].Name)
	})

	return pairs, nil
}

func getLogConfiguration(service types.ServiceConfig, project *types.Project) *ecs.TaskDefinition_LogConfiguration {
	options := map[string]string{
		"awslogs-region":        cloudformation.Ref("AWS::Region"),
		"awslogs-group":         cloudformation.Ref("LogGroup"),
		"awslogs-stream-prefix": project.Name,
	}
	if service.Logging != nil {
		for k, v := range service.Logging.Options {
			if strings.HasPrefix(k, "awslogs-") {
				options[k] = v
			}
		}
	}
	logConfiguration := &ecs.TaskDefinition_LogConfiguration{
		LogDriver: ecsapi.LogDriverAwslogs,
		Options:   options,
	}
	return logConfiguration
}

func toSystemControls(sysctls types.Mapping) []ecs.TaskDefinition_SystemControl {
	sys := []ecs.TaskDefinition_SystemControl{}
	for k, v := range sysctls {
		sys = append(sys, ecs.TaskDefinition_SystemControl{
			Namespace: cloudformation.String(k),
			Value:     cloudformation.String(v),
		})
	}
	return sys
}

const miB = 1024 * 1024

func toLimits(service types.ServiceConfig) (string, string, error) {
	mem, cpu, err := getConfiguredLimits(service)
	if err != nil {
		return "", "", err
	}
	if requireEC2(service) {
		// just return configured limits expressed in Mb and CPU units
		var cpuLimit, memLimit string
		if cpu > 0 {
			cpuLimit = fmt.Sprint(cpu)
		}
		if mem > 0 {
			memLimit = fmt.Sprint(mem / miB)
		}
		return cpuLimit, memLimit, nil
	}

	// All possible cpu/mem values for Fargate
	fargateCPUToMem := map[int64][]types.UnitBytes{
		256:  {512, 1024, 2048},
		512:  {1024, 2048, 3072, 4096},
		1024: {2048, 3072, 4096, 5120, 6144, 7168, 8192},
		2048: {4096, 5120, 6144, 7168, 8192, 9216, 10240, 11264, 12288, 13312, 14336, 15360, 16384},
		4096: {8192, 9216, 10240, 11264, 12288, 13312, 14336, 15360, 16384, 17408, 18432, 19456, 20480, 21504, 22528, 23552, 24576, 25600, 26624, 27648, 28672, 29696, 30720},
	}
	cpuLimit := "256"
	memLimit := "512"
	if mem == 0 && cpu == 0 {
		return cpuLimit, memLimit, nil
	}

	var cpus []int64
	for k := range fargateCPUToMem {
		cpus = append(cpus, k)
	}
	sort.Slice(cpus, func(i, j int) bool { return cpus[i] < cpus[j] })

	for _, fargateCPU := range cpus {
		options := fargateCPUToMem[fargateCPU]
		if cpu <= fargateCPU {
			for _, m := range options {
				if mem <= m*miB {
					cpuLimit = strconv.FormatInt(fargateCPU, 10)
					memLimit = strconv.FormatInt(int64(m), 10)
					return cpuLimit, memLimit, nil
				}
			}
		}
	}
	return "", "", fmt.Errorf("the resources requested are not supported by ECS/Fargate")
}

func getConfiguredLimits(service types.ServiceConfig) (types.UnitBytes, int64, error) {
	if service.Deploy == nil {
		return 0, 0, nil
	}

	limits := service.Deploy.Resources.Limits
	if limits == nil {
		limits = service.Deploy.Resources.Reservations
	}
	if limits == nil {
		return 0, 0, nil
	}

	if limits.NanoCPUs == "" {
		return limits.MemoryBytes, 0, nil
	}
	v, err := opts.ParseCPUs(limits.NanoCPUs)
	if err != nil {
		return 0, 0, err
	}

	return limits.MemoryBytes, v / 1e6, nil
}

func toContainerReservation(service types.ServiceConfig) (string, int) {
	cpuReservation := ".0"
	memReservation := 0

	if service.Deploy == nil {
		return cpuReservation, memReservation
	}

	reservations := service.Deploy.Resources.Reservations
	if reservations == nil {
		return cpuReservation, memReservation
	}
	return reservations.NanoCPUs, int(reservations.MemoryBytes / miB)
}

func toPlacementConstraints(deploy *types.DeployConfig) []ecs.TaskDefinition_TaskDefinitionPlacementConstraint {
	if deploy == nil || deploy.Placement.Constraints == nil || len(deploy.Placement.Constraints) == 0 {
		return nil
	}
	pl := []ecs.TaskDefinition_TaskDefinitionPlacementConstraint{}
	for _, c := range deploy.Placement.Constraints {
		pl = append(pl, ecs.TaskDefinition_TaskDefinitionPlacementConstraint{
			Expression: cloudformation.String(c),
			Type:       "",
		})
	}
	return pl
}

func toPortMappings(ports []types.ServicePortConfig) []ecs.TaskDefinition_PortMapping {
	if len(ports) == 0 {
		return nil
	}
	m := []ecs.TaskDefinition_PortMapping{}
	for _, p := range ports {
		m = append(m, ecs.TaskDefinition_PortMapping{
			ContainerPort: cloudformation.Int(int(p.Target)),
			HostPort:      cloudformation.Int(int(p.Published)),
			Protocol:      cloudformation.String(p.Protocol),
		})
	}
	return m
}

func toUlimits(ulimits map[string]*types.UlimitsConfig) []ecs.TaskDefinition_Ulimit {
	if len(ulimits) == 0 {
		return nil
	}
	u := []ecs.TaskDefinition_Ulimit{}
	for k, v := range ulimits {
		u = append(u, ecs.TaskDefinition_Ulimit{
			Name:      k,
			SoftLimit: v.Soft,
			HardLimit: v.Hard,
		})
	}
	return u
}

func toLinuxParameters(service types.ServiceConfig) *ecs.TaskDefinition_LinuxParameters {
	return &ecs.TaskDefinition_LinuxParameters{
		Capabilities:       toKernelCapabilities(service.CapAdd, service.CapDrop),
		Devices:            nil,
		InitProcessEnabled: cloudformation.Bool(service.Init != nil && *service.Init),
		MaxSwap:            cloudformation.Int(0),
		// FIXME SharedMemorySize:   service.ShmSize,
		Swappiness: cloudformation.Int(0),
		Tmpfs:      toTmpfs(service.Tmpfs),
	}
}

func toTmpfs(tmpfs types.StringList) []ecs.TaskDefinition_Tmpfs {
	if len(tmpfs) == 0 {
		return nil
	}
	o := []ecs.TaskDefinition_Tmpfs{}
	for _, path := range tmpfs {
		o = append(o, ecs.TaskDefinition_Tmpfs{
			ContainerPath: cloudformation.String(path),
			Size:          100, // size is required on ECS, unlimited by the compose spec
		})
	}
	return o
}

func toKernelCapabilities(add []string, drop []string) *ecs.TaskDefinition_KernelCapabilities {
	if len(add) == 0 && len(drop) == 0 {
		return nil
	}
	return &ecs.TaskDefinition_KernelCapabilities{
		Add:  add,
		Drop: drop,
	}

}

func toHealthCheck(check *types.HealthCheckConfig) *ecs.TaskDefinition_HealthCheck {
	if check == nil {
		return nil
	}
	retries := 0
	if check.Retries != nil {
		retries = int(*check.Retries)
	}
	return &ecs.TaskDefinition_HealthCheck{
		Command:     check.Test,
		Interval:    cloudformation.Int(durationToInt(check.Interval)),
		Retries:     cloudformation.Int(retries),
		StartPeriod: cloudformation.Int(durationToInt(check.StartPeriod)),
		Timeout:     cloudformation.Int(durationToInt(check.Timeout)),
	}
}

func durationToInt(interval *types.Duration) int {
	if interval == nil {
		return 0
	}
	v := int(time.Duration(*interval).Seconds())
	return v
}

func toHostEntryPtr(hosts types.HostsList) []ecs.TaskDefinition_HostEntry {
	if len(hosts) == 0 {
		return nil
	}
	e := []ecs.TaskDefinition_HostEntry{}
	for _, h := range hosts {
		parts := strings.SplitN(h, ":", 2) // FIXME this should be handled by compose-go
		e = append(e, ecs.TaskDefinition_HostEntry{
			Hostname:  cloudformation.String(parts[0]),
			IpAddress: cloudformation.String(parts[1]),
		})
	}
	return e
}

func getRepoCredentials(service types.ServiceConfig) *ecs.TaskDefinition_RepositoryCredentials {
	if value, ok := service.Extensions[extensionPullCredentials]; ok {
		return &ecs.TaskDefinition_RepositoryCredentials{CredentialsParameter: cloudformation.String(value.(string))}
	}
	return nil
}

func requireEC2(s types.ServiceConfig) bool {
	return gpuRequirements(s) > 0
}

func gpuRequirements(s types.ServiceConfig) int64 {
	if deploy := s.Deploy; deploy != nil {
		if reservations := deploy.Resources.Reservations; reservations != nil {
			for _, resource := range reservations.GenericResources {
				if resource.DiscreteResourceSpec.Kind == "gpus" {
					return resource.DiscreteResourceSpec.Value
				}
			}
			for _, device := range reservations.Devices {
				if len(device.Capabilities) == 1 && device.Capabilities[0] == "gpu" {
					return device.Count
				}
			}
		}
	}
	return 0
}
