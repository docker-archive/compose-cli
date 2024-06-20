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
	"regexp"
	"strings"

	ecsapi "github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	cloudmapapi "github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/awslabs/goformation/v4/cloudformation"
	"github.com/awslabs/goformation/v4/cloudformation/ec2"
	"github.com/awslabs/goformation/v4/cloudformation/ecs"
	"github.com/awslabs/goformation/v4/cloudformation/elasticloadbalancingv2"
	"github.com/awslabs/goformation/v4/cloudformation/iam"
	"github.com/awslabs/goformation/v4/cloudformation/logs"
	"github.com/awslabs/goformation/v4/cloudformation/secretsmanager"
	cloudmap "github.com/awslabs/goformation/v4/cloudformation/servicediscovery"
	"github.com/cnabio/cnab-to-oci/remotes"
	"github.com/compose-spec/compose-go/types"
	"github.com/distribution/distribution/v3/reference"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/opencontainers/go-digest"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"

	"github.com/docker/compose-cli/api/config"
	"github.com/docker/compose-cli/utils"
)

func (b *ecsAPIService) Convert(ctx context.Context, project *types.Project, options api.ConvertOptions) ([]byte, error) {
	if err := checkUnsupportedConvertOptions(ctx, options); err != nil {
		return nil, err
	}
	err := b.resolveServiceImagesDigests(ctx, project)
	if err != nil {
		return nil, err
	}

	template, err := b.convert(ctx, project)
	if err != nil {
		return nil, err
	}

	bytes, err := marshall(template, options.Format)
	if err != nil {
		return nil, err
	}

	x, ok := project.Extensions[extensionCloudFormation]
	if !ok {
		return bytes, nil
	}
	if options.Format != "yaml" {
		return nil, fmt.Errorf("format %q with overlays is not supported", options.Format)
	}

	nodes, err := yaml.Parse(string(bytes))
	if err != nil {
		return nil, err
	}

	bytes, err = yaml.Marshal(x)
	if err != nil {
		return nil, err
	}
	overlay, err := yaml.Parse(string(bytes))
	if err != nil {
		return nil, err
	}
	nodes, err = merge2.Merge(overlay, nodes, yaml.MergeOptions{
		ListIncreaseDirection: yaml.MergeOptionsListPrepend,
	})
	if err != nil {
		return nil, err
	}

	s, err := nodes.String()
	if err != nil {
		return nil, err
	}
	bytes = []byte(s)
	return bytes, err
}

func checkUnsupportedConvertOptions(ctx context.Context, o api.ConvertOptions) error {
	return utils.CheckUnsupported(ctx, nil, o.Output, "", "convert", "output")
}

func (b *ecsAPIService) resolveServiceImagesDigests(ctx context.Context, project *types.Project) error {
	configFile, err := cliconfig.Load(config.Dir())
	if err != nil {
		return err
	}

	resolver := remotes.CreateResolver(configFile)
	return project.ResolveImages(func(named reference.Named) (digest.Digest, error) {
		_, desc, err := resolver.Resolve(ctx, named.String())
		return desc.Digest, err
	})
}

func (b *ecsAPIService) convert(ctx context.Context, project *types.Project) (*cloudformation.Template, error) {
	err := b.checkCompatibility(project)
	if err != nil {
		return nil, err
	}

	template := cloudformation.NewTemplate()
	resources, err := b.parse(ctx, project, template)
	if err != nil {
		return nil, err
	}

	err = b.ensureResources(&resources, project, template)
	if err != nil {
		return nil, err
	}

	for name, secret := range project.Secrets {
		err := b.createSecret(project, name, secret, template)
		if err != nil {
			return nil, err
		}
	}

	b.createLogGroup(project, template)

	// Private DNS namespace will allow DNS name for the services to be <service>.<project>.local
	b.createCloudMap(project, template, resources.vpc)

	b.createNFSMountTarget(project, resources, template)

	b.createAccessPoints(project, resources, template)

	for _, service := range project.Services {
		err := b.createService(project, service, template, resources)
		if err != nil {
			return nil, err
		}

		err = b.createAutoscalingPolicy(project, resources, template, service)
		if err != nil {
			return nil, err
		}
	}

	err = b.createCapacityProvider(ctx, project, template, resources)
	if err != nil {
		return nil, err
	}

	return template, nil
}

func (b *ecsAPIService) createService(project *types.Project, service types.ServiceConfig, template *cloudformation.Template, resources awsResources) error {
	taskExecutionRole := b.createTaskExecutionRole(project, service, template)
	taskRole, err := b.createTaskRole(project, service, template, resources)
	if err != nil {
		return err
	}

	definition, err := b.createTaskDefinition(project, service, resources)
	if err != nil {
		return err
	}
	definition.ExecutionRoleArn = cloudformation.Ref(taskExecutionRole)
	if taskRole != "" {
		definition.TaskRoleArn = cloudformation.Ref(taskRole)
	}

	taskDefinition := fmt.Sprintf("%sTaskDefinition", normalizeResourceName(service.Name))
	template.Resources[taskDefinition] = definition

	var healthCheck *cloudmap.Service_HealthCheckConfig
	serviceRegistry := b.createServiceRegistry(service, template, healthCheck)

	var (
		dependsOn []string
		serviceLB []ecs.Service_LoadBalancer
	)

	for _, port := range service.Ports {
		lbSecurityGroupName := serviceIngressSecGroupName(service.Name, true)
		serviceSecurityGroupName := serviceIngressSecGroupName(service.Name, false)
		// outside to ingress
		b.createIngress(service, lbSecurityGroupName, port, template, resources)
		// ingress to service
		b.createCrossGroupIngress(service, lbSecurityGroupName, serviceSecurityGroupName, port, template, resources)
		// TODO attach new secgroup to this service
		protocol := strings.ToUpper(port.Protocol)
		if resources.loadBalancerType == elbv2.LoadBalancerTypeEnumApplication {
			// we don't set Https as a certificate must be specified for HTTPS listeners
			protocol = elbv2.ProtocolEnumHttp
		}
		targetGroupName := b.createTargetGroup(project, service, port, template, protocol, resources.vpc)
		listenerName := b.createListener(service, port, template, targetGroupName, resources.loadBalancer, protocol)
		dependsOn = append(dependsOn, listenerName)
		serviceLB = append(serviceLB, ecs.Service_LoadBalancer{
			ContainerName:  service.Name,
			ContainerPort:  int(port.Target),
			TargetGroupArn: cloudformation.Ref(targetGroupName),
		})
	}

	desiredCount := 1
	if service.Deploy != nil && service.Deploy.Replicas != nil {
		desiredCount = int(*service.Deploy.Replicas)
	}

	for dependency := range service.DependsOn {
		dependsOn = append(dependsOn, serviceResourceName(dependency))
	}

	for _, s := range service.Volumes {
		dependsOn = append(dependsOn, b.mountTargets(s.Source, resources)...)
	}

	minPercent, maxPercent, err := computeRollingUpdateLimits(service)
	if err != nil {
		return err
	}

	assignPublicIP := ecsapi.AssignPublicIpEnabled
	launchType := ecsapi.LaunchTypeFargate
	platformVersion := "1.4.0" // LATEST which is set to 1.3.0 (?) which doesn’t allow efs volumes.
	if requireEC2(service) {
		assignPublicIP = ecsapi.AssignPublicIpDisabled
		launchType = ecsapi.LaunchTypeEc2
		platformVersion = "" // The platform version must be null when specifying an EC2 launch type
	}

	template.Resources[serviceResourceName(service.Name)] = &ecs.Service{
		AWSCloudFormationDependsOn: dependsOn,
		Cluster:                    resources.cluster.ARN(),
		DesiredCount:               desiredCount,
		DeploymentController: &ecs.Service_DeploymentController{
			Type: ecsapi.DeploymentControllerTypeEcs,
		},
		DeploymentConfiguration: &ecs.Service_DeploymentConfiguration{
			MaximumPercent:        maxPercent,
			MinimumHealthyPercent: minPercent,
		},
		LaunchType: launchType,
		// TODO we miss support for https://github.com/aws/containers-roadmap/issues/631 to select a capacity provider
		LoadBalancers: serviceLB,
		NetworkConfiguration: &ecs.Service_NetworkConfiguration{
			AwsvpcConfiguration: &ecs.Service_AwsVpcConfiguration{
				AssignPublicIp: assignPublicIP,
				SecurityGroups: resources.serviceSecurityGroups(service),
				Subnets:        resources.subnetsIDs(),
			},
		},
		PlatformVersion:    platformVersion,
		PropagateTags:      ecsapi.PropagateTagsService,
		SchedulingStrategy: ecsapi.SchedulingStrategyReplica,
		ServiceRegistries:  []ecs.Service_ServiceRegistry{serviceRegistry},
		Tags:               serviceTags(project, service),
		TaskDefinition:     cloudformation.Ref(normalizeResourceName(taskDefinition)),
	}
	return nil
}

const allProtocols = "-1"

func (b *ecsAPIService) createIngress(service types.ServiceConfig, net string, port types.ServicePortConfig, template *cloudformation.Template, resources awsResources) {
	protocol := strings.ToUpper(port.Protocol)
	if protocol == "" {
		protocol = allProtocols
	}
	ingress := fmt.Sprintf("%s%dIngress", normalizeResourceName(net), port.Target)
	template.Resources[ingress] = &ec2.SecurityGroupIngress{
		CidrIp:      "0.0.0.0/0",
		Description: fmt.Sprintf("%s:%d/%s on %s network", service.Name, port.Target, port.Protocol, net),
		GroupId:     resources.securityGroups[net],
		FromPort:    int(port.Target),
		IpProtocol:  protocol,
		ToPort:      int(port.Target),
	}
}

func (b *ecsAPIService) createCrossGroupIngress(service types.ServiceConfig, source string, dest string, port types.ServicePortConfig, template *cloudformation.Template, resources awsResources) {
	protocol := strings.ToUpper(port.Protocol)
	if protocol == "" {
		protocol = allProtocols
	}
	ingress := fmt.Sprintf("%s%d%sIngress", normalizeResourceName(source), port.Target, normalizeResourceName(dest))
	template.Resources[ingress] = &ec2.SecurityGroupIngress{
		SourceSecurityGroupId: resources.securityGroups[source],
		Description:           fmt.Sprintf("LB connectivity on %d", port.Target),
		GroupId:               resources.securityGroups[dest],
		FromPort:              int(port.Target),
		IpProtocol:            protocol,
		ToPort:                int(port.Target),
	}
}

func (b *ecsAPIService) createSecret(project *types.Project, name string, s types.SecretConfig, template *cloudformation.Template) error {
	if s.External.External {
		return nil
	}
	sensitiveData, err := os.ReadFile(s.File)
	if err != nil {
		return err
	}

	resource := fmt.Sprintf("%sSecret", normalizeResourceName(s.Name))
	template.Resources[resource] = &secretsmanager.Secret{
		Description:  fmt.Sprintf("Secret %s", s.Name),
		SecretString: string(sensitiveData),
		Tags:         projectTags(project),
	}
	s.Name = cloudformation.Ref(resource)
	project.Secrets[name] = s
	return nil
}

func (b *ecsAPIService) createLogGroup(project *types.Project, template *cloudformation.Template) {
	retention := 0
	if v, ok := project.Extensions[extensionRetention]; ok {
		retention = v.(int)
	}
	logGroup := fmt.Sprintf("/docker-compose/%s", project.Name)
	template.Resources["LogGroup"] = &logs.LogGroup{
		LogGroupName:    logGroup,
		RetentionInDays: retention,
	}
}

func computeRollingUpdateLimits(service types.ServiceConfig) (int, int, error) {
	maxPercent := 200
	minPercent := 100
	if service.Deploy == nil || service.Deploy.UpdateConfig == nil {
		return minPercent, maxPercent, nil
	}
	updateConfig := service.Deploy.UpdateConfig
	min, okMin := updateConfig.Extensions[extensionMinPercent]
	if okMin {
		minPercent = min.(int)
	}
	max, okMax := updateConfig.Extensions[extensionMaxPercent]
	if okMax {
		maxPercent = max.(int)
	}
	if okMin && okMax {
		return minPercent, maxPercent, nil
	}

	if updateConfig.Parallelism != nil {
		parallelism := int(*updateConfig.Parallelism)
		if service.Deploy.Replicas == nil {
			return minPercent, maxPercent,
				fmt.Errorf("rolling update configuration require deploy.replicas to be set")
		}
		replicas := int(*service.Deploy.Replicas)
		if replicas < parallelism {
			return minPercent, maxPercent,
				fmt.Errorf("deploy.replicas (%d) must be greater than deploy.update_config.parallelism (%d)", replicas, parallelism)
		}
		if !okMin {
			minPercent = (replicas - parallelism) * 100 / replicas
		}
		if !okMax {
			maxPercent = (replicas + parallelism) * 100 / replicas
		}
	}
	return minPercent, maxPercent, nil
}

func (b *ecsAPIService) createListener(service types.ServiceConfig, port types.ServicePortConfig,
	template *cloudformation.Template,
	targetGroupName string, loadBalancer awsResource, protocol string) string {
	listenerName := fmt.Sprintf(
		"%s%s%dListener",
		normalizeResourceName(service.Name),
		strings.ToUpper(port.Protocol),
		port.Target,
	)
	// add listener to dependsOn
	// https://stackoverflow.com/questions/53971873/the-target-group-does-not-have-an-associated-load-balancer
	template.Resources[listenerName] = &elasticloadbalancingv2.Listener{
		DefaultActions: []elasticloadbalancingv2.Listener_Action{
			{
				ForwardConfig: &elasticloadbalancingv2.Listener_ForwardConfig{
					TargetGroups: []elasticloadbalancingv2.Listener_TargetGroupTuple{
						{
							TargetGroupArn: cloudformation.Ref(targetGroupName),
						},
					},
				},
				Type: elbv2.ActionTypeEnumForward,
			},
		},
		LoadBalancerArn: loadBalancer.ARN(),
		Protocol:        protocol,
		Port:            int(port.Target),
	}
	return listenerName
}

func (b *ecsAPIService) createTargetGroup(project *types.Project, service types.ServiceConfig, port types.ServicePortConfig, template *cloudformation.Template, protocol string, vpc string) string {
	targetGroupName := fmt.Sprintf(
		"%s%s%dTargetGroup",
		normalizeResourceName(service.Name),
		strings.ToUpper(port.Protocol),
		port.Published,
	)
	template.Resources[targetGroupName] = &elasticloadbalancingv2.TargetGroup{
		Port:       int(port.Target),
		Protocol:   protocol,
		Tags:       projectTags(project),
		TargetType: elbv2.TargetTypeEnumIp,
		VpcId:      vpc,
	}
	return targetGroupName
}

func (b *ecsAPIService) createServiceRegistry(service types.ServiceConfig, template *cloudformation.Template, healthCheck *cloudmap.Service_HealthCheckConfig) ecs.Service_ServiceRegistry {
	serviceRegistration := fmt.Sprintf("%sServiceDiscoveryEntry", normalizeResourceName(service.Name))
	serviceRegistry := ecs.Service_ServiceRegistry{
		RegistryArn: cloudformation.GetAtt(serviceRegistration, "Arn"),
	}

	template.Resources[serviceRegistration] = &cloudmap.Service{
		Description:       fmt.Sprintf("%q service discovery entry in Cloud Map", service.Name),
		HealthCheckConfig: healthCheck,
		HealthCheckCustomConfig: &cloudmap.Service_HealthCheckCustomConfig{
			FailureThreshold: 1,
		},
		Name:        service.Name,
		NamespaceId: cloudformation.Ref("CloudMap"),
		DnsConfig: &cloudmap.Service_DnsConfig{
			DnsRecords: []cloudmap.Service_DnsRecord{
				{
					TTL:  60,
					Type: cloudmapapi.RecordTypeA,
				},
			},
			RoutingPolicy: cloudmapapi.RoutingPolicyMultivalue,
		},
	}
	return serviceRegistry
}

func (b *ecsAPIService) createTaskExecutionRole(project *types.Project, service types.ServiceConfig, template *cloudformation.Template) string {
	taskExecutionRole := fmt.Sprintf("%sTaskExecutionRole", normalizeResourceName(service.Name))
	policies := b.createPolicies(project, service)
	template.Resources[taskExecutionRole] = &iam.Role{
		AssumeRolePolicyDocument: ecsTaskAssumeRolePolicyDocument,
		Policies:                 policies,
		ManagedPolicyArns: []string{
			ecsTaskExecutionPolicy,
			ecrReadOnlyPolicy,
		},
		Tags: serviceTags(project, service),
	}
	return taskExecutionRole
}

func (b *ecsAPIService) createTaskRole(project *types.Project, service types.ServiceConfig, template *cloudformation.Template, resources awsResources) (string, error) {
	taskRole := fmt.Sprintf("%sTaskRole", normalizeResourceName(service.Name))
	rolePolicies := []iam.Role_Policy{}
	if roles, ok := service.Extensions[extensionRole]; ok {
		rolePolicies = append(rolePolicies, iam.Role_Policy{
			PolicyName:     fmt.Sprintf("%sPolicy", normalizeResourceName(service.Name)),
			PolicyDocument: roles,
		})
	}
	for _, vol := range service.Volumes {
		if vol.Source == "" {
			return "", fmt.Errorf(
				"service %s has an invalid volume %s: ECS does not support sourceless volumes",
				service.Name,
				vol.Target,
			)
		}
		rolePolicies = append(rolePolicies, iam.Role_Policy{
			PolicyName:     fmt.Sprintf("%s%sVolumeMountPolicy", normalizeResourceName(service.Name), normalizeResourceName(vol.Source)),
			PolicyDocument: volumeMountPolicyDocument(vol.Source, resources.filesystems[vol.Source].ARN()),
		})
	}
	managedPolicies := []string{}
	if v, ok := service.Extensions[extensionManagedPolicies]; ok {
		for _, s := range v.([]interface{}) {
			managedPolicies = append(managedPolicies, s.(string))
		}
	}
	if len(rolePolicies) == 0 && len(managedPolicies) == 0 {
		return "", nil
	}
	template.Resources[taskRole] = &iam.Role{
		AssumeRolePolicyDocument: ecsTaskAssumeRolePolicyDocument,
		Policies:                 rolePolicies,
		ManagedPolicyArns:        managedPolicies,
		Tags:                     serviceTags(project, service),
	}
	return taskRole, nil
}

func (b *ecsAPIService) createCloudMap(project *types.Project, template *cloudformation.Template, vpc string) {
	template.Resources["CloudMap"] = &cloudmap.PrivateDnsNamespace{
		Description: fmt.Sprintf("Service Map for Docker Compose project %s", project.Name),
		Name:        fmt.Sprintf("%s.local", project.Name),
		Vpc:         vpc,
	}
}

func (b *ecsAPIService) createPolicies(project *types.Project, service types.ServiceConfig) []iam.Role_Policy {
	var arns []string
	if value, ok := service.Extensions[extensionPullCredentials]; ok {
		arns = append(arns, value.(string))
	}
	for _, secret := range service.Secrets {
		arns = append(arns, project.Secrets[secret.Source].Name)
	}
	if len(arns) > 0 {
		return []iam.Role_Policy{
			{
				PolicyDocument: &PolicyDocument{
					Statement: []PolicyStatement{
						{
							Effect:   "Allow",
							Action:   []string{actionGetSecretValue, actionGetParameters, actionDecrypt},
							Resource: arns,
						},
					},
				},
				PolicyName: fmt.Sprintf("%sGrantAccessToSecrets", service.Name),
			},
		}
	}
	return nil
}

func networkResourceName(network string) string {
	return fmt.Sprintf("%sNetwork", normalizeResourceName(network))
}

func serviceIngressSecGroupName(service string, isLBSide bool) string {
	var header string
	if isLBSide {
		header = "LB"
	} else {
		header = "Service"
	}
	return fmt.Sprintf("%s%sIngressSecurityGroup", normalizeResourceName(service), header)
}

func serviceResourceName(service string) string {
	return fmt.Sprintf("%sService", normalizeResourceName(service))
}

func volumeResourceName(service string) string {
	return fmt.Sprintf("%sFilesystem", normalizeResourceName(service))
}

func normalizeResourceName(s string) string {
	//nolint:staticcheck // Preserving for compatibility
	return strings.Title(regexp.MustCompile("[^a-zA-Z0-9]+").ReplaceAllString(s, ""))
}
