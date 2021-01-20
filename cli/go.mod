module github.com/docker/compose-cli/cli

go 1.15

replace (
	github.com/docker/compose-cli/aci => ../aci
	github.com/docker/compose-cli/api => ../api
	github.com/docker/compose-cli/ecs => ../ecs
	github.com/docker/compose-cli/kube => ../kube
	github.com/docker/compose-cli/local => ../local

	// the distribution version from ecs plugin is quite old and it breaks containerd
	// we need to create a new release tag for docker/distribution
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20200708230824-53e18a9d9bfe

	// (for buildx)
	github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305
)

require (
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Microsoft/go-winio v0.4.16
	github.com/compose-spec/compose-go v0.0.0-20210119095023-cd294eea46e9
	github.com/containerd/console v1.0.1
	github.com/containerd/containerd v1.4.3
	github.com/docker/cli v20.10.2+incompatible
	github.com/docker/compose-cli/aci v0.0.0-00010101000000-000000000000
	github.com/docker/compose-cli/api v0.0.0-00010101000000-000000000000
	github.com/docker/compose-cli/ecs v0.0.0-00010101000000-000000000000
	github.com/docker/compose-cli/kube v0.0.0-00010101000000-000000000000
	github.com/docker/compose-cli/local v0.0.0-00010101000000-000000000000
	github.com/docker/docker v20.10.2+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.35.0
	google.golang.org/protobuf v1.25.0
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.0.3
)
