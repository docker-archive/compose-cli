module github.com/docker/compose-cli

go 1.15

require (
	github.com/AlecAivazis/survey/v2 v2.2.3
	github.com/Azure/azure-sdk-for-go v48.2.0+incompatible
	github.com/Azure/azure-storage-file-go v0.8.0
	github.com/Azure/go-autorest/autorest v0.11.12
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.3
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.2
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Microsoft/go-winio v0.4.15
	github.com/aws/aws-sdk-go v1.35.33
	github.com/awslabs/goformation/v4 v4.15.6
	github.com/buger/goterm v0.0.0-20200322175922-2f3e71b85129
	github.com/cnabio/cnab-to-oci v0.3.1-beta1
	github.com/compose-spec/compose-go v0.0.0-20210119095023-cd294eea46e9
	github.com/containerd/console v1.0.1
	github.com/containerd/containerd v1.4.3
	github.com/containerd/continuity v0.0.0-20200928162600-f2cc35102c2a // indirect
	github.com/docker/buildx v0.5.1
	github.com/docker/cli v20.10.1+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.1+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/gobwas/httphead v0.0.0-20180130184737-2c6c146eadee // indirect
	github.com/gobwas/pool v0.2.0 // indirect
	github.com/gobwas/ws v1.0.4
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/iancoleman/strcase v0.1.2
	github.com/joho/godotenv v1.3.0
	github.com/moby/buildkit v0.8.1-0.20201205083753-0af7b1b9c693
	github.com/moby/term v0.0.0-20201110203204-bea5bbe245bf
	github.com/morikuni/aec v1.0.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/tsdb v0.10.0
	github.com/sanathkr/go-yaml v0.0.0-20170819195128-ed9d249f429b
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	golang.org/x/oauth2 v0.0.0-20201109201403-9fd604954f58
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	google.golang.org/grpc v1.33.2
	google.golang.org/protobuf v1.25.0
	gopkg.in/ini.v1 v1.62.0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.0.3
	helm.sh/helm/v3 v3.5.0
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
	sigs.k8s.io/kustomize/kyaml v0.10.5
)

replace (
	// the distribution version from ecs plugin is quite old and it breaks containerd
	// we need to create a new release tag for docker/distribution
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20200708230824-53e18a9d9bfe

	// (for buildx)
	github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305
)
