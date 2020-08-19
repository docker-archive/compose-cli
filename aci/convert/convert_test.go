/*
   Copyright 2020 Docker, Inc.

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

package convert

import (
	"context"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/containerinstance/mgmt/containerinstance"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/compose-spec/compose-go/types"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/docker/api/containers"
	"github.com/docker/api/context/store"
)

var convertCtx = store.AciContext{
	SubscriptionID: "subID",
	ResourceGroup:  "rg",
	Location:       "eu",
}

func TestProjectName(t *testing.T) {
	project := types.Project{
		Name: "TEST",
	}
	containerGroup, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)
	assert.Equal(t, *containerGroup.Name, "test")
}

func TestContainerGroupToContainer(t *testing.T) {
	myContainerGroup := containerinstance.ContainerGroup{
		ContainerGroupProperties: &containerinstance.ContainerGroupProperties{
			IPAddress: &containerinstance.IPAddress{
				Ports: &[]containerinstance.Port{{
					Port: to.Int32Ptr(80),
				}},
				IP: to.StringPtr("42.42.42.42"),
			},
			OsType: "Linux",
		},
	}
	myContainer := containerinstance.Container{
		Name: to.StringPtr("myContainerID"),
		ContainerProperties: &containerinstance.ContainerProperties{
			Image:   to.StringPtr("sha256:666"),
			Command: to.StringSlicePtr([]string{"mycommand"}),
			Ports: &[]containerinstance.ContainerPort{{
				Port: to.Int32Ptr(80),
			}},
			EnvironmentVariables: nil,
			InstanceView: &containerinstance.ContainerPropertiesInstanceView{
				RestartCount: nil,
				CurrentState: &containerinstance.ContainerState{
					State: to.StringPtr("Running"),
				},
			},
			Resources: &containerinstance.ResourceRequirements{
				Limits: &containerinstance.ResourceLimits{
					CPU:        to.Float64Ptr(3),
					MemoryInGB: to.Float64Ptr(0.1),
				},
			},
		},
	}

	var expectedContainer = containers.Container{
		ID:          "myContainerID",
		Status:      "Running",
		Image:       "sha256:666",
		Command:     "mycommand",
		CPULimit:    3,
		MemoryLimit: 107374182,
		Platform:    "Linux",
		Ports: []containers.Port{{
			HostPort:      uint32(80),
			ContainerPort: uint32(80),
			Protocol:      "tcp",
			HostIP:        "42.42.42.42",
		}},
		RestartPolicyCondition: "any",
	}

	container := ContainerGroupToContainer("myContainerID", myContainerGroup, myContainer)
	assert.DeepEqual(t, container, expectedContainer)
}

func TestComposeContainerGroupToContainerWithDnsSideCarSide(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
			},
			{
				Name:  "service2",
				Image: "image2",
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(*group.Containers, 3))

	assert.Equal(t, *(*group.Containers)[0].Name, "service1")
	assert.Equal(t, *(*group.Containers)[1].Name, "service2")
	assert.Equal(t, *(*group.Containers)[2].Name, ComposeDNSSidecarName)

	assert.DeepEqual(t, *(*group.Containers)[2].Command, []string{"sh", "-c", "echo 127.0.0.1 service1 >> /etc/hosts;echo 127.0.0.1 service2 >> /etc/hosts;sleep infinity"})

	assert.Equal(t, *(*group.Containers)[0].Image, "image1")
	assert.Equal(t, *(*group.Containers)[1].Image, "image2")
	assert.Equal(t, *(*group.Containers)[2].Image, dnsSidecarImage)
}

func TestComposeSingleContainerGroupToContainerNoDnsSideCarSide(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)

	assert.Assert(t, is.Len(*group.Containers, 1))
	assert.Equal(t, *(*group.Containers)[0].Name, "service1")
	assert.Equal(t, *(*group.Containers)[0].Image, "image1")
}

func TestComposeSingleContainerRestartPolicy(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
				Deploy: &types.DeployConfig{
					RestartPolicy: &types.RestartPolicy{
						Condition: "on-failure",
					},
				},
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)

	assert.Assert(t, is.Len(*group.Containers, 1))
	assert.Equal(t, *(*group.Containers)[0].Name, "service1")
	assert.Equal(t, group.RestartPolicy, containerinstance.OnFailure)
}

func TestComposeMultiContainerRestartPolicy(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
				Deploy: &types.DeployConfig{
					RestartPolicy: &types.RestartPolicy{
						Condition: "on-failure",
					},
				},
			},
			{
				Name:  "service2",
				Image: "image2",
				Deploy: &types.DeployConfig{
					RestartPolicy: &types.RestartPolicy{
						Condition: "on-failure",
					},
				},
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)

	assert.Assert(t, is.Len(*group.Containers, 3))
	assert.Equal(t, *(*group.Containers)[0].Name, "service1")
	assert.Equal(t, group.RestartPolicy, containerinstance.OnFailure)
	assert.Equal(t, *(*group.Containers)[1].Name, "service2")
	assert.Equal(t, group.RestartPolicy, containerinstance.OnFailure)
}

func TestComposeInconsistentMultiContainerRestartPolicy(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
				Deploy: &types.DeployConfig{
					RestartPolicy: &types.RestartPolicy{
						Condition: "any",
					},
				},
			},
			{
				Name:  "service2",
				Image: "image2",
				Deploy: &types.DeployConfig{
					RestartPolicy: &types.RestartPolicy{
						Condition: "on-failure",
					},
				},
			},
		},
	}

	_, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.Error(t, err, "ACI integration does not support specifying different restart policies on containers in the same compose application")
}

func TestLabelsErrorMessage(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
				Labels: map[string]string{
					"label1": "value1",
				},
			},
		},
	}

	_, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.Error(t, err, "ACI integration does not support labels in compose applications")
}

func TestComposeSingleContainerGroupToContainerDefaultRestartPolicy(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)

	assert.Assert(t, is.Len(*group.Containers, 1))
	assert.Equal(t, *(*group.Containers)[0].Name, "service1")
	assert.Equal(t, group.RestartPolicy, containerinstance.Always)
}

func TestComposeContainerGroupToContainerMultiplePorts(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
				Ports: []types.ServicePortConfig{
					{
						Published: 80,
						Target:    80,
					},
				},
			},
			{
				Name:  "service2",
				Image: "image2",
				Ports: []types.ServicePortConfig{
					{
						Published: 8080,
						Target:    8080,
					},
				},
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(*group.Containers, 3))

	container1 := (*group.Containers)[0]
	assert.Equal(t, *container1.Name, "service1")
	assert.Equal(t, *container1.Image, "image1")
	assert.Equal(t, *(*container1.Ports)[0].Port, int32(80))

	container2 := (*group.Containers)[1]
	assert.Equal(t, *container2.Name, "service2")
	assert.Equal(t, *container2.Image, "image2")
	assert.Equal(t, *(*container2.Ports)[0].Port, int32(8080))

	groupPorts := *group.IPAddress.Ports
	assert.Assert(t, is.Len(groupPorts, 2))
	assert.Equal(t, *groupPorts[0].Port, int32(80))
	assert.Equal(t, *groupPorts[1].Port, int32(8080))
}

func TestComposeContainerGroupToContainerResourceLimits(t *testing.T) {
	_0_1Gb := 0.1 * 1024 * 1024 * 1024
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
				Deploy: &types.DeployConfig{
					Resources: types.Resources{
						Limits: &types.Resource{
							NanoCPUs:    "0.1",
							MemoryBytes: types.UnitBytes(_0_1Gb),
						},
					},
				},
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)

	limits := *((*group.Containers)[0]).Resources.Limits
	assert.Equal(t, *limits.CPU, float64(0.1))
	assert.Equal(t, *limits.MemoryInGB, float64(0.1))
}

func TestComposeContainerGroupToContainerResourceLimitsDefaults(t *testing.T) {
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
				Deploy: &types.DeployConfig{
					Resources: types.Resources{
						Limits: &types.Resource{
							NanoCPUs:    "",
							MemoryBytes: 0,
						},
					},
				},
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)

	limits := *((*group.Containers)[0]).Resources.Limits
	assert.Equal(t, *limits.CPU, float64(1))
	assert.Equal(t, *limits.MemoryInGB, float64(1))
}

func TestComposeContainerGroupToContainerenvVar(t *testing.T) {
	err := os.Setenv("key2", "value2")
	assert.NilError(t, err)
	project := types.Project{
		Services: []types.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
				Environment: types.MappingWithEquals{
					"key1": to.StringPtr("value1"),
					"key2": nil,
				},
			},
		},
	}

	group, err := ToContainerGroup(context.TODO(), convertCtx, project)
	assert.NilError(t, err)

	envVars := *((*group.Containers)[0]).EnvironmentVariables
	assert.Assert(t, is.Len(envVars, 2))
	assert.Assert(t, is.Contains(envVars, containerinstance.EnvironmentVariable{Name: to.StringPtr("key1"), Value: to.StringPtr("value1")}))
	assert.Assert(t, is.Contains(envVars, containerinstance.EnvironmentVariable{Name: to.StringPtr("key2"), Value: to.StringPtr("value2")}))
}

func TestConvertToAciRestartPolicyCondition(t *testing.T) {
	assert.Equal(t, toAciRestartPolicy("none"), containerinstance.Never)
	assert.Equal(t, toAciRestartPolicy("always"), containerinstance.Always)
	assert.Equal(t, toAciRestartPolicy("on-failure"), containerinstance.OnFailure)
	assert.Equal(t, toAciRestartPolicy("on-failure:5"), containerinstance.Always)
}

func TestConvertToDockerRestartPolicyCondition(t *testing.T) {
	assert.Equal(t, toContainerRestartPolicy(containerinstance.Never), "none")
	assert.Equal(t, toContainerRestartPolicy(containerinstance.Always), "any")
	assert.Equal(t, toContainerRestartPolicy(containerinstance.OnFailure), "on-failure")
	assert.Equal(t, toContainerRestartPolicy(""), "any")
}

func TestConvertContainerGroupStatus(t *testing.T) {
	assert.Equal(t, "Running", GetStatus(container(to.StringPtr("Running")), group(to.StringPtr("Started"))))
	assert.Equal(t, "Terminated", GetStatus(container(to.StringPtr("Terminated")), group(to.StringPtr("Stopped"))))
	assert.Equal(t, "Node Stopped", GetStatus(container(nil), group(to.StringPtr("Stopped"))))
	assert.Equal(t, "Node Started", GetStatus(container(nil), group(to.StringPtr("Started"))))

	assert.Equal(t, "Running", GetStatus(container(to.StringPtr("Running")), group(nil)))
	assert.Equal(t, "Unknown", GetStatus(container(nil), group(nil)))
}

func container(status *string) containerinstance.Container {
	var state *containerinstance.ContainerState = nil
	if status != nil {
		state = &containerinstance.ContainerState{
			State: status,
		}
	}
	return containerinstance.Container{
		ContainerProperties: &containerinstance.ContainerProperties{
			InstanceView: &containerinstance.ContainerPropertiesInstanceView{
				CurrentState: state,
			},
		},
	}
}

func group(status *string) containerinstance.ContainerGroup {
	var view *containerinstance.ContainerGroupPropertiesInstanceView = nil
	if status != nil {
		view = &containerinstance.ContainerGroupPropertiesInstanceView{
			State: status,
		}
	}
	return containerinstance.ContainerGroup{
		ContainerGroupProperties: &containerinstance.ContainerGroupProperties{
			InstanceView: view,
		},
	}
}
