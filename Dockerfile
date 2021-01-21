# syntax=docker/dockerfile:experimental


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

ARG GO_VERSION=1.15.6-alpine
ARG GOLANGCI_LINT_VERSION=v1.33.0-alpine
ARG PROTOC_GEN_GO_VERSION=v1.4.3

FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS base
WORKDIR /compose-cli
ENV GO111MODULE=on
RUN apk add --no-cache \
    git \
    docker \
    make \
    protoc \
    protobuf-dev
COPY api/go.* api/
COPY local/go.* local/
COPY ecs/go.* ecs/
COPY aci/go.* aci/
COPY kube/go.* kube/
COPY cli/go.* cli/
COPY builder.Makefile .

RUN --mount=type=cache,target=/go/pkg/mod \
    make -f builder.Makefile go-mod-download

FROM base AS make-protos
ARG PROTOC_GEN_GO_VERSION
RUN go get github.com/golang/protobuf/protoc-gen-go@${PROTOC_GEN_GO_VERSION}
COPY . .
RUN make -f builder.Makefile protos

FROM golangci/golangci-lint:${GOLANGCI_LINT_VERSION} AS lint-base

FROM base AS lint
ENV CGO_ENABLED=0
COPY --from=lint-base /usr/bin/golangci-lint /usr/bin/golangci-lint
ARG BUILD_TAGS
ARG GIT_TAG
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.cache/golangci-lint \
    BUILD_TAGS=${BUILD_TAGS} \
    GIT_TAG=${GIT_TAG} \
    make -f builder.Makefile lint

FROM base AS make-cli
ENV CGO_ENABLED=0
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_TAGS
ARG GIT_TAG
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    BUILD_TAGS=${BUILD_TAGS} \
    GIT_TAG=${GIT_TAG} \
    make BINARY=/out/docker -f builder.Makefile cli

FROM base AS make-cross
ARG BUILD_TAGS
ARG GIT_TAG
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    BUILD_TAGS=${BUILD_TAGS} \
    GIT_TAG=${GIT_TAG} \
    make BINARY=/out/docker  -f builder.Makefile cross

FROM scratch AS protos
COPY --from=make-protos /compose-cli/cli/server/protos .

FROM scratch AS cli
COPY --from=make-cli /out/* .

FROM scratch AS cross
COPY --from=make-cross /out/* .

FROM base AS test
ENV CGO_ENABLED=0
ARG BUILD_TAGS
ARG GIT_TAG
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    BUILD_TAGS=${BUILD_TAGS} \
    GIT_TAG=${GIT_TAG} \
    make -f builder.Makefile test

FROM base AS check-license-headers
RUN go get -u github.com/kunalkushwaha/ltag
RUN --mount=target=. \
    make -f builder.Makefile check-license-headers

FROM base AS make-go-mod-tidy
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    make -f builder.Makefile go-mod-tidy

FROM scratch AS go-mod-tidy
COPY --from=make-go-mod-tidy /compose-cli/api/go.mod api/
COPY --from=make-go-mod-tidy /compose-cli/api/go.sum api/
COPY --from=make-go-mod-tidy /compose-cli/local/go.mod local/
COPY --from=make-go-mod-tidy /compose-cli/local/go.sum local/
COPY --from=make-go-mod-tidy /compose-cli/ecs/go.mod ecs/
COPY --from=make-go-mod-tidy /compose-cli/ecs/go.sum ecs/
COPY --from=make-go-mod-tidy /compose-cli/aci/go.mod aci/
COPY --from=make-go-mod-tidy /compose-cli/aci/go.sum aci/
COPY --from=make-go-mod-tidy /compose-cli/kube/go.mod kube/
COPY --from=make-go-mod-tidy /compose-cli/kube/go.sum kube/
COPY --from=make-go-mod-tidy /compose-cli/cli/go.mod cli/
COPY --from=make-go-mod-tidy /compose-cli/cli/go.sum cli/

FROM base AS check-go-mod
COPY . .
RUN make -f builder.Makefile check-go-mod
