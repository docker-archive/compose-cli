#   Copyright 2020 The 2020 Docker, Inc.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

export DOCKER_BUILDKIT=1

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	MOBY_DOCKER=/usr/bin/docker
	SED_I=sed -i
endif
ifeq ($(UNAME_S),Darwin)
	MOBY_DOCKER=/Applications/Docker.app/Contents/Resources/bin/docker
	SED_I=sed -i ''
endif

GIT_TAG=$(shell git describe --tags --match "v[0-9]*" --abbrev=0 )
PREVIOUS_TAG=$(shell git describe --tags --match "v[0-9]*" --abbrev=0 ${GIT_TAG}^)

all: cli

protos: ## Generate go code from .proto files
	@docker build . --target protos \
	--output ./protos

cli: ## Compile the cli
	@docker build . --target cli \
	--platform local \
	--build-arg BUILD_TAGS=example,local \
	--output ./bin

e2e-local: ## Run End to end local tests
	go test -v ./tests/e2e ./tests/skip-win-ci-e2e ./local/e2e

e2e-win-ci: ## Run End to end local tests on windows CI, no docker for linux containers available ATM
	go test -v ./tests/e2e

e2e-aci: ## Run End to end ACI tests (requires azure login)
	go test -v ./tests/aci-e2e

cross: ## Compile the CLI for linux, darwin and windows
	@docker build . --target cross \
	--build-arg BUILD_TAGS \
	--output ./bin \

test: ## Run unit tests
	@docker build . \
	--build-arg BUILD_TAGS=example,local \
	--target test

cache-clear: ## Clear the builder cache
	@docker builder prune --force --filter type=exec.cachemount --filter=unused-for=24h

lint: ## run linter(s)
	@docker build . --target lint

serve: cli ## start server
	@./bin/docker serve --address unix:///tmp/backend.sock

moby-cli-link: ## create com.docker.cli symlink if does not already exist
	ln -s $(MOBY_DOCKER) /usr/local/bin/com.docker.cli

changelog-release: ## generate CHNAGELOG.MD for last tagged release
	rm -f CHANGELOG.md
	docker --context default  run --rm -v $(shell pwd):/usr/local/src/your-app ferrarimarco/github-changelog-generator --user docker --project api --token ${GITHUB_TOKEN} --no-issues --since-tag ${PREVIOUS_TAG} --no-unreleased --header-label "" --no-compare-link
	${SED_I} '1,4d' CHANGELOG.md # remove changelog tag header, we should make a PR in github-changelog-generator to add this option

changelog-preview: ## generate CHNAGELOG.MD for all unreleased things since last TAG
	rm -f CHANGELOG.md
	docker --context default  run --rm -v $(shell pwd):/usr/local/src/your-app ferrarimarco/github-changelog-generator --user docker --project api --token ${GITHUB_TOKEN} --no-issues --unreleased-only --header-label "" --no-compare-link
	${SED_I} '1,4d' CHANGELOG.md # remove changelog tag header, we should make a PR in github-changelog-generator to add this option

help: ## Show help
	@echo Please specify a build target. The choices are:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

FORCE:

.PHONY: all protos cli e2e-local cross test cache-clear lint serve classic-link help changelog-release changelog-preview
