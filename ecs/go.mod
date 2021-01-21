module github.com/docker/compose-cli/ecs

go 1.15

replace (
	github.com/docker/compose-cli/api => ../api

	github.com/docker/compose-cli/local => ../local

	// the distribution version from ecs plugin is quite old and it breaks containerd
	// we need to create a new release tag for docker/distribution
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20200708230824-53e18a9d9bfe

	// (for buildx)
	github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305
)

require (
	github.com/AlecAivazis/survey/v2 v2.2.7
	github.com/aws/aws-sdk-go v1.36.28
	github.com/awslabs/goformation/v4 v4.15.7
	github.com/cnabio/cnab-to-oci v0.3.1-beta1
	github.com/compose-spec/compose-go v0.0.0-20210119095023-cd294eea46e9
	github.com/docker/cli v20.10.2+incompatible
	github.com/docker/compose-cli/api v0.0.0-00010101000000-000000000000
	github.com/docker/compose-cli/local v0.0.0-00010101000000-000000000000
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.2+incompatible
	github.com/docker/go-units v0.4.0
	github.com/golang/mock v1.4.4
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-uuid v1.0.1
	github.com/iancoleman/strcase v0.1.3
	github.com/joho/godotenv v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/sanathkr/go-yaml v0.0.0-20170819195128-ed9d249f429b
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	gopkg.in/ini.v1 v1.62.0
	gotest.tools/v3 v3.0.3
	sigs.k8s.io/kustomize/kyaml v0.10.6
)
