module github.com/docker/compose-cli/aci

go 1.15

replace github.com/docker/compose-cli/api => ../api

require (
	github.com/AlecAivazis/survey/v2 v2.2.7
	github.com/Azure/azure-sdk-for-go v50.1.0+incompatible
	github.com/Azure/azure-storage-file-go v0.8.0
	github.com/Azure/go-autorest/autorest v0.11.17
	github.com/Azure/go-autorest/autorest/adal v0.9.10
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.6
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.2
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/buger/goterm v0.0.0-20200322175922-2f3e71b85129
	github.com/compose-spec/compose-go v0.0.0-20210119095023-cd294eea46e9
	github.com/docker/cli v20.10.2+incompatible
	github.com/docker/compose-cli/api v0.0.0-00010101000000-000000000000
	github.com/docker/docker v20.10.2+incompatible
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.0.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-uuid v1.0.2
	github.com/morikuni/aec v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/tsdb v0.10.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/oauth2 v0.0.0-20210113205817-d3ed898aa8a3
	gotest.tools/v3 v3.0.3
)
