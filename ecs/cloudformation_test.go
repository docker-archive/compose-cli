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
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/awslabs/goformation/v4/cloudformation"
	"github.com/awslabs/goformation/v4/cloudformation/ec2"
	"github.com/awslabs/goformation/v4/cloudformation/ecs"
	"github.com/awslabs/goformation/v4/cloudformation/efs"
	"github.com/awslabs/goformation/v4/cloudformation/elasticloadbalancingv2"
	"github.com/awslabs/goformation/v4/cloudformation/iam"
	"github.com/awslabs/goformation/v4/cloudformation/logs"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/golang/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestSimpleConvert(t *testing.T) {
	bytes, err := os.ReadFile("testdata/input/simple-single-service.yaml")
	assert.NilError(t, err)
	template := convertYaml(t, string(bytes), nil, useDefaultVPC)
	resultAsJSON, err := marshall(template, "yaml")
	assert.NilError(t, err)
	result := fmt.Sprintf("%s\n", string(resultAsJSON))
	expected := "simple-cloudformation-conversion.golden"
	golden.Assert(t, result, expected)
}

func TestSlightlyComplexConvert(t *testing.T) {
	bytes, err := ioutil.ReadFile("testdata/input/slightly-complex-service.yaml")
	assert.NilError(t, err)
	template := convertYaml(t, string(bytes), nil, useDefaultVPC)
	resultAsJSON, err := marshall(template, "yaml")
	assert.NilError(t, err)
	result := fmt.Sprintf("%s\n", string(resultAsJSON))
	expected := "slightly-complex-cloudformation-conversion.golden"
	golden.Assert(t, result, expected)
}

func TestLogging(t *testing.T) {
	template := convertYaml(t, `
services:
  foo:
    image: hello_world
    logging:
      options:
        awslogs-datetime-pattern: "FOO"

x-aws-logs_retention: 10
`, nil, useDefaultVPC)
	def := template.Resources["FooTaskDefinition"].(*ecs.TaskDefinition)
	logging := getMainContainer(def, t).LogConfiguration
	if logging != nil {
		assert.Equal(t, logging.Options["awslogs-datetime-pattern"], "FOO")
	} else {
		t.Fatal("Logging not configured")
	}

	logGroup := template.Resources["LogGroup"].(*logs.LogGroup)
	assert.Equal(t, logGroup.RetentionInDays, 10)
}

func TestEnvFile(t *testing.T) {
	template := convertYaml(t, `
services:
  foo:
    image: hello_world
    env_file:
      - testdata/input/envfile
`, nil, useDefaultVPC)
	def := template.Resources["FooTaskDefinition"].(*ecs.TaskDefinition)
	env := getMainContainer(def, t).Environment
	var found bool
	for _, pair := range env {
		if pair.Name == "FOO" {
			assert.Equal(t, pair.Value, "BAR")
			found = true
		}
	}
	assert.Check(t, found, "environment variable FOO not set")
}

func TestEnvFileAndEnv(t *testing.T) {
	template := convertYaml(t, `
services:
  foo:
    image: hello_world
    env_file:
      - testdata/input/envfile
    environment:
      - "FOO=ZOT"
`, nil, useDefaultVPC)
	def := template.Resources["FooTaskDefinition"].(*ecs.TaskDefinition)
	env := getMainContainer(def, t).Environment
	var found bool
	for _, pair := range env {
		if pair.Name == "FOO" {
			assert.Equal(t, pair.Value, "ZOT")
			found = true
		}
	}
	assert.Check(t, found, "environment variable FOO not set")
}

func TestRollingUpdateLimits(t *testing.T) {
	template := convertYaml(t, `
services:
  foo:
    image: hello_world
    deploy:
      replicas: 4 
      update_config:
        parallelism: 2
`, nil, useDefaultVPC)
	service := template.Resources["FooService"].(*ecs.Service)
	assert.Check(t, service.DeploymentConfiguration.MaximumPercent == 150)
	assert.Check(t, service.DeploymentConfiguration.MinimumHealthyPercent == 50)
}

func TestRollingUpdateExtension(t *testing.T) {
	template := convertYaml(t, `
services:
  foo:
    image: hello_world
    deploy:
      update_config:
        x-aws-min_percent: 25
        x-aws-max_percent: 125
`, nil, useDefaultVPC)
	service := template.Resources["FooService"].(*ecs.Service)
	assert.Check(t, service.DeploymentConfiguration.MaximumPercent == 125)
	assert.Check(t, service.DeploymentConfiguration.MinimumHealthyPercent == 25)
}

func TestRolePolicy(t *testing.T) {
	template := convertYaml(t, `
services:
  foo:
    image: hello_world
    x-aws-pull_credentials: "secret"
`, nil, useDefaultVPC)
	x := template.Resources["FooTaskExecutionRole"]
	assert.Check(t, x != nil)
	role := *(x.(*iam.Role))
	assert.Check(t, role.ManagedPolicyArns[0] == ecsTaskExecutionPolicy)
	assert.Check(t, role.ManagedPolicyArns[1] == ecrReadOnlyPolicy)
	// We expect an extra policy has been created for x-aws-pull_credentials
	assert.Check(t, len(role.Policies) == 1)
	policy := role.Policies[0].PolicyDocument.(*PolicyDocument)
	expected := []string{"secretsmanager:GetSecretValue", "ssm:GetParameters", "kms:Decrypt"}
	assert.DeepEqual(t, expected, policy.Statement[0].Action)
	assert.DeepEqual(t, []string{"secret"}, policy.Statement[0].Resource)
}

func TestMapNetworksToSecurityGroups(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: hello_world
    networks:
      - front-tier
      - back-tier

networks:
  front-tier:
    name: public
  back-tier:
    internal: true
`, nil, useDefaultVPC)
	assert.Check(t, template.Resources["FronttierNetwork"] != nil)
	assert.Check(t, template.Resources["BacktierNetwork"] != nil)
	assert.Check(t, template.Resources["BacktierNetworkIngress"] != nil)
	i := template.Resources["FronttierNetworkIngress"]
	assert.Check(t, i != nil)
	ingress := *i.(*ec2.SecurityGroupIngress)
	assert.Check(t, ingress.SourceSecurityGroupId == cloudformation.Ref("FronttierNetwork"))

}

func TestLoadBalancerTypeApplication(t *testing.T) {
	cases := []string{
		`services:
  test:
    image: nginx
    ports:
      - 80:80
`,
		`services:
  test:
    image: nginx
    ports:
      - target: 8080
        x-aws-protocol: http
`,
	}
	for _, y := range cases {
		template := convertYaml(t, y, nil, useDefaultVPC)
		lb := template.Resources["LoadBalancer"]
		assert.Check(t, lb != nil)
		loadBalancer := *lb.(*elasticloadbalancingv2.LoadBalancer)
		assert.Check(t, len(loadBalancer.Name) <= 32)
		assert.Check(t, loadBalancer.Type == elbv2.LoadBalancerTypeEnumApplication)
		assert.Check(t, len(loadBalancer.SecurityGroups) > 0)
	}
}

func TestNoLoadBalancerIfNoPortExposed(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
  foo:
    image: bar
`, nil, useDefaultVPC)
	for _, r := range template.Resources {
		assert.Check(t, r.AWSCloudFormationType() != "AWS::ElasticLoadBalancingV2::TargetGroup")
		assert.Check(t, r.AWSCloudFormationType() != "AWS::ElasticLoadBalancingV2::Listener")
		assert.Check(t, r.AWSCloudFormationType() != "AWS::ElasticLoadBalancingV2::PortPublisher")
	}
}

func TestServiceReplicas(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
    deploy:
      replicas: 10
`, nil, useDefaultVPC)
	s := template.Resources["TestService"]
	assert.Check(t, s != nil)
	service := *s.(*ecs.Service)
	assert.Check(t, service.DesiredCount == 10)
}

func TestTaskSizeConvert(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
`, nil, useDefaultVPC)
	def := template.Resources["TestTaskDefinition"].(*ecs.TaskDefinition)
	assert.Equal(t, def.Cpu, "256")
	assert.Equal(t, def.Memory, "512")

	template = convertYaml(t, `
services:
  test:
    image: nginx
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 2048M
`, nil, useDefaultVPC)
	def = template.Resources["TestTaskDefinition"].(*ecs.TaskDefinition)
	assert.Equal(t, def.Cpu, "512")
	assert.Equal(t, def.Memory, "2048")

	template = convertYaml(t, `
services:
  test:
    image: nginx
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8192M
`, nil, useDefaultVPC)
	def = template.Resources["TestTaskDefinition"].(*ecs.TaskDefinition)
	assert.Equal(t, def.Cpu, "4096")
	assert.Equal(t, def.Memory, "8192")

	template = convertYaml(t, `
services:
  test:
    image: nginx
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 792Mb
        reservations:
          generic_resources: 
            - discrete_resource_spec:
                kind: gpus
                value: 2
`, nil, useDefaultVPC, useGPU)
	def = template.Resources["TestTaskDefinition"].(*ecs.TaskDefinition)
	assert.Equal(t, def.Cpu, "4000")
	assert.Equal(t, def.Memory, "792")

	template = convertYaml(t, `
services:
  test:
    image: nginx
    deploy:
      resources:
        reservations:
          generic_resources: 
            - discrete_resource_spec:
                kind: gpus
                value: 2
`, nil, useDefaultVPC, useGPU)
	def = template.Resources["TestTaskDefinition"].(*ecs.TaskDefinition)
	assert.Equal(t, def.Cpu, "")
	assert.Equal(t, def.Memory, "")

	template = convertYaml(t, `
services:
  test:
    image: nginx
    deploy:
      resources:
        reservations:
          devices: 
            - capabilities: [gpu]
              count: 2
`, nil, useDefaultVPC, useGPU)
	def = template.Resources["TestTaskDefinition"].(*ecs.TaskDefinition)
	assert.Equal(t, def.Cpu, "")
	assert.Equal(t, def.Memory, "")
}

func TestLoadBalancerTypeNetwork(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
    ports:
      - 80:80
      - 88:88
`, nil, useDefaultVPC)
	lb := template.Resources["LoadBalancer"]
	assert.Check(t, lb != nil)
	loadBalancer := *lb.(*elasticloadbalancingv2.LoadBalancer)
	assert.Check(t, loadBalancer.Type == elbv2.LoadBalancerTypeEnumNetwork)
}

func TestUseExternalNetwork(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
networks:
  default:
    external: true
    name: sg-123abc
`, nil, useDefaultVPC, func(m *MockAPIMockRecorder) {
		m.SecurityGroupExists(gomock.Any(), "sg-123abc").Return(true, nil)
	})
	assert.Check(t, template.Resources["DefaultNetwork"] == nil)
	assert.Check(t, template.Resources["DefaultNetworkIngress"] == nil)
	s := template.Resources["TestService"].(*ecs.Service)
	assert.Check(t, s != nil)                                                                    //nolint:staticcheck
	assert.Check(t, s.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups[0] == "sg-123abc") //nolint:staticcheck
}

func TestUseExternalVolume(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
volumes:
  db-data:
    external: true
    name: fs-123abc
`, nil, useDefaultVPC, func(m *MockAPIMockRecorder) {
		m.ResolveFileSystem(gomock.Any(), "fs-123abc").Return(existingAWSResource{id: "fs-123abc"}, nil)
	})
	s := template.Resources["DbdataNFSMountTargetOnSubnet1"].(*efs.MountTarget)
	assert.Check(t, s != nil)                    //nolint:staticcheck
	assert.Equal(t, s.FileSystemId, "fs-123abc") //nolint:staticcheck

	s = template.Resources["DbdataNFSMountTargetOnSubnet2"].(*efs.MountTarget)
	assert.Check(t, s != nil)                    //nolint:staticcheck
	assert.Equal(t, s.FileSystemId, "fs-123abc") //nolint:staticcheck
}

func TestCreateVolume(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
volumes:
  db-data: 
    driver_opts:
        backup_policy: ENABLED
        lifecycle_policy: AFTER_30_DAYS
        performance_mode: maxIO
        throughput_mode: provisioned
        provisioned_throughput: 1024
`, nil, useDefaultVPC, func(m *MockAPIMockRecorder) {
		m.ListFileSystems(gomock.Any(), map[string]string{
			api.ProjectLabel: t.Name(),
			api.VolumeLabel:  "db-data",
		}).Return(nil, nil)
	})
	n := volumeResourceName("db-data")
	f := template.Resources[n].(*efs.FileSystem)
	assert.Check(t, f != nil)                                               //nolint:staticcheck
	assert.Equal(t, f.BackupPolicy.Status, "ENABLED")                       //nolint:staticcheck
	assert.Equal(t, f.LifecyclePolicies[0].TransitionToIA, "AFTER_30_DAYS") //nolint:staticcheck
	assert.Equal(t, f.PerformanceMode, "maxIO")                             //nolint:staticcheck
	assert.Equal(t, f.ThroughputMode, "provisioned")                        //nolint:staticcheck
	assert.Equal(t, f.ProvisionedThroughputInMibps, float64(1024))          //nolint:staticcheck

	s := template.Resources["DbdataNFSMountTargetOnSubnet1"].(*efs.MountTarget)
	assert.Check(t, s != nil)                              //nolint:staticcheck
	assert.Equal(t, s.FileSystemId, cloudformation.Ref(n)) //nolint:staticcheck
}

func TestCreateSourcelessVolume(t *testing.T) {
	convertYaml(t, `
services:
  test:
    image: nginx
    volumes:
      - db-data
volumes:
    db-data:
`,
		errors.New("service test has an invalid volume db-data: ECS does not support sourceless volumes"),
		useDefaultVPC, func(m *MockAPIMockRecorder) {
			m.ListFileSystems(gomock.Any(), map[string]string{
				api.ProjectLabel: t.Name(),
				api.VolumeLabel:  "db-data",
			}).Return(nil, nil)
		})
}

func TestCreateAccessPoint(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
volumes:
  db-data:
    driver_opts:
      uid: 1002
      gid: 1002
`, nil, useDefaultVPC, func(m *MockAPIMockRecorder) {
		m.ListFileSystems(gomock.Any(), gomock.Any()).Return(nil, nil)
	})
	a := template.Resources["DbdataAccessPoint"].(*efs.AccessPoint)
	assert.Check(t, a != nil)                //nolint:staticcheck
	assert.Equal(t, a.PosixUser.Uid, "1002") //nolint:staticcheck
	assert.Equal(t, a.PosixUser.Gid, "1002") //nolint:staticcheck
}

func TestReusePreviousVolume(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
volumes:
  db-data: {}
`, nil, useDefaultVPC, func(m *MockAPIMockRecorder) {
		m.ListFileSystems(gomock.Any(), map[string]string{
			api.ProjectLabel: t.Name(),
			api.VolumeLabel:  "db-data",
		}).Return([]awsResource{
			existingAWSResource{
				id: "fs-123abc",
			},
		}, nil)
	})
	s := template.Resources["DbdataNFSMountTargetOnSubnet1"].(*efs.MountTarget)
	assert.Check(t, s != nil)                    //nolint:staticcheck
	assert.Equal(t, s.FileSystemId, "fs-123abc") //nolint:staticcheck
}

func TestServiceMapping(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: "image"
    command: "command"
    entrypoint: "entrypoint"
    environment:
      - "FOO=BAR"
    cap_add:
      - SYS_PTRACE
    cap_drop:
      - SYSLOG
    init: true
    user: "user"
    working_dir: "working_dir"
`, nil, useDefaultVPC)
	def := template.Resources["TestTaskDefinition"].(*ecs.TaskDefinition)
	container := getMainContainer(def, t)
	assert.Equal(t, container.Image, "image")
	assert.Equal(t, container.Command[0], "command")
	assert.Equal(t, container.EntryPoint[0], "entrypoint")
	assert.Equal(t, get(container.Environment, "FOO"), "BAR")
	assert.Check(t, container.LinuxParameters.InitProcessEnabled)
	assert.Equal(t, container.LinuxParameters.Capabilities.Add[0], "SYS_PTRACE")
	assert.Equal(t, container.LinuxParameters.Capabilities.Drop[0], "SYSLOG")
	assert.Equal(t, container.User, "user")
	assert.Equal(t, container.WorkingDirectory, "working_dir")
}

func get(l []ecs.TaskDefinition_KeyValuePair, name string) string {
	for _, e := range l {
		if e.Name == name {
			return e.Value
		}
	}
	return ""
}

func TestResourcesHaveProjectTagSet(t *testing.T) {
	template := convertYaml(t, `
services:
  test:
    image: nginx
    ports:
      - 80:80
      - 88:88
`, nil, useDefaultVPC)
	for _, r := range template.Resources {
		tags := reflect.Indirect(reflect.ValueOf(r)).FieldByName("Tags")
		if !tags.IsValid() {
			continue
		}
		for i := 0; i < tags.Len(); i++ {
			k := tags.Index(i).FieldByName("Key").String()
			v := tags.Index(i).FieldByName("Value").String()
			if k == api.ProjectLabel {
				assert.Equal(t, v, t.Name())
			}
		}
	}
}

func TestTemplateMetadata(t *testing.T) {
	template := convertYaml(t, `
x-aws-cluster: "arn:aws:ecs:region:account:cluster/name"
services:
  test:
    image: nginx
`, nil, useDefaultVPC, func(m *MockAPIMockRecorder) {
		m.ResolveCluster(gomock.Any(), "arn:aws:ecs:region:account:cluster/name").Return(existingAWSResource{
			arn: "arn:aws:ecs:region:account:cluster/name",
			id:  "name",
		}, nil)
	})
	assert.Equal(t, template.Metadata["Cluster"], "arn:aws:ecs:region:account:cluster/name")
}

func TestARNUsedAsVpcID(t *testing.T) {
	convertYaml(t, `
x-aws-vpc: "arn:aws:ec2:us-west-1:EXAMPLE:vpc/vpc-1234acbd"
services:
  test:
    image: nginx
`, nil, func(m *MockAPIMockRecorder) {
		m.CheckVPC(gomock.Any(), "vpc-1234acbd").Return(nil)
		m.GetSubNets(gomock.Any(), "vpc-1234acbd").Return([]awsResource{
			existingAWSResource{id: "subnet1"},
			existingAWSResource{id: "subnet2"},
		}, nil)
		m.IsPublicSubnet(gomock.Any(), "subnet1").Return(true, nil)
		m.IsPublicSubnet(gomock.Any(), "subnet2").Return(true, nil)
	})
}

func convertYaml(t *testing.T, yaml string, assertErr error, fn ...func(m *MockAPIMockRecorder)) *cloudformation.Template {
	project := loadConfig(t, yaml)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockAPI(ctrl)
	for _, f := range fn {
		f(m.EXPECT())
	}

	backend := &ecsAPIService{
		aws: m,
	}
	template, err := backend.convert(context.TODO(), project)
	if assertErr == nil {
		assert.NilError(t, err)
	} else {
		assert.Error(t, err, assertErr.Error())
	}

	return template
}

func loadConfig(t *testing.T, yaml string) *types.Project {
	dict, err := loader.ParseYAML([]byte(yaml))
	assert.NilError(t, err)
	model, err := loader.Load(types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{Config: dict},
		},
	}, func(options *loader.Options) {
		options.Name = t.Name()
	})
	assert.NilError(t, err)
	return model
}

func getMainContainer(def *ecs.TaskDefinition, t *testing.T) ecs.TaskDefinition_ContainerDefinition {
	for _, c := range def.ContainerDefinitions {
		if c.Essential {
			return c
		}
	}
	t.Fail()
	return def.ContainerDefinitions[0]
}

func useDefaultVPC(m *MockAPIMockRecorder) {
	m.GetDefaultVPC(gomock.Any()).Return("vpc-123", nil)
	m.GetSubNets(gomock.Any(), "vpc-123").Return([]awsResource{
		existingAWSResource{id: "subnet1"},
		existingAWSResource{id: "subnet2"},
	}, nil)
	m.IsPublicSubnet(gomock.Any(), "subnet1").Return(true, nil)
	m.IsPublicSubnet(gomock.Any(), "subnet2").Return(true, nil)
}

func useGPU(m *MockAPIMockRecorder) {
	m.GetParameter(gomock.Any(), gomock.Any()).Return("", nil)
}
