#   Copyright 2020 Docker Compose CLI authors

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

PKG_NAME:=github.com/docker/compose-cli

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

CROSS_PLATFORMS:=darwin/amd64 linux/amd64 linux/arm linux/arm64 windows/amd64

PROTOS=$(shell find protos -name \*.proto)

EXTENSION:=
ifeq ($(GOOS),windows)
  EXTENSION:=.exe
endif

STATIC_FLAGS=CGO_ENABLED=0

GIT_TAG?=$(shell git describe --tags --match "v[0-9]*")

LDFLAGS="-s -w -X $(PKG_NAME)/internal.Version=${GIT_TAG}"
GO_BUILD=$(STATIC_FLAGS) go build -trimpath -ldflags=$(LDFLAGS)

BINARY?=bin/docker
BINARY_WITH_EXTENSION=$(BINARY)$(EXTENSION)

WORK_DIR:=$(shell mktemp -d)

TAGS:=
ifdef BUILD_TAGS
  TAGS=-tags $(BUILD_TAGS)
  LINT_TAGS=--build-tags $(BUILD_TAGS)
endif

TAR_TRANSFORM:=--transform s/packaging/docker/ --transform s/bin/docker/ --transform s/docker-.*/docker/
ifneq ($(findstring bsd,$(shell tar --version)),)
  TAR_TRANSFORM=-s /packaging/docker/ -s /bin/docker/ -s /docker-.*/docker/
endif

all: cli

.PHONY: protos
protos:
	protoc -I. --go_out=plugins=grpc,paths=source_relative:. ${PROTOS}

.PHONY: cli
cli:
	GOOS=${GOOS} GOARCH=${GOARCH} $(GO_BUILD) $(TAGS) -o $(BINARY_WITH_EXTENSION) ./cli

cross-%:
	GOOS=$(word 1,$(subst -, ,$(*))) GOARCH=$(word 2,$(subst -, ,$(*))) \
	$(GO_BUILD) $(TAGS) -o $(BINARY)-$(*)$(if $(filter windows,$(word 1,$(subst -, ,$(*)))),.exe,) ./cli;

.PHONY: cross
cross: $(foreach plat,$(CROSS_PLATFORMS),cross-$(subst /,-,$(plat)))

.PHONY: test
test:
	go test $(TAGS) -cover $(shell go list ./... | grep -vE 'e2e')

.PHONY: lint
lint:
	golangci-lint run $(LINT_TAGS) --timeout 10m0s ./...

.PHONY: import-restrictions
import-restrictions:
	import-restrictions --configuration import-restrictions.yaml

.PHONY: check-licese-headers
check-license-headers:
	./scripts/validate/fileheader

.PHONY: check-go-mod
check-go-mod:
	./scripts/validate/check-go-mod

.PHONY: package
package: cross
	mkdir -p dist
	$(foreach plat,$(filter-out windows%,$(CROSS_PLATFORMS)), \
	tar czf dist/docker-$(subst /,-,$(plat)).tar.gz $(TAR_TRANSFORM) packaging/LICENSE $(BINARY)-$(subst /,-,$(plat));)
	cp $(BINARY)-windows-amd64.exe $(WORK_DIR)/docker.exe
	rm -f dist/docker-windows-amd64.zip && zip dist/docker-windows-amd64.zip -j packaging/LICENSE $(WORK_DIR)/docker.exe
	rm -r $(WORK_DIR)
