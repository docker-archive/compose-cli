module github.com/docker/compose-cli/local

go 1.15

replace github.com/docker/compose-cli/api => ../api

require (
	github.com/compose-spec/compose-go v0.0.0-20210119095023-cd294eea46e9
	github.com/docker/buildx v0.5.1
	github.com/docker/cli v20.10.2+incompatible
	github.com/docker/compose-cli/api v0.0.0-00010101000000-000000000000
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.2+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/moby/buildkit v0.8.1-0.20201205083753-0af7b1b9c693
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/sanathkr/go-yaml v0.0.0-20170819195128-ed9d249f429b
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.0.3
)
