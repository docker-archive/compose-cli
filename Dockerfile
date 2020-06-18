# syntax = docker/dockerfile:experimental
ARG GO_VERSION=1.14.3-alpine
ARG GOLANGCI_LINT_VERSION=v1.27.0-alpine

FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS base
WORKDIR /api
ENV GO111MODULE=on
ENV GOPRIVATE=github.com/docker/ecs-plugin
RUN apk add --no-cache \
    docker \
    make \
    protoc \
    protobuf-dev
COPY go.* .
RUN go mod download

FROM base AS make-protos
RUN go get github.com/golang/protobuf/protoc-gen-go@v1.4.1
COPY . .
RUN make -f builder.Makefile protos

FROM golangci/golangci-lint:${GOLANGCI_LINT_VERSION} AS lint-base

FROM base AS lint
ENV CGO_ENABLED=0
COPY --from=lint-base /usr/bin/golangci-lint /usr/bin/golangci-lint
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.cache/golangci-lint \
    make -f builder.Makefile lint

FROM base AS make-cli
ENV CGO_ENABLED=0
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_TAGS
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=ssh \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    BUILD_TAGS=${BUILD_TAGS} \
    make BINARY=/out/docker -f builder.Makefile cli

FROM base AS make-cross
ARG BUILD_TAGS
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    BUILD_TAGS=${BUILD_TAGS} \
    make BINARY=/out/docker  -f builder.Makefile cross

FROM scratch AS protos
COPY --from=make-protos /api/protos .

FROM scratch AS cli
COPY --from=make-cli /out/* .

FROM scratch AS cross
COPY --from=make-cross /out/* .

FROM base as test
ARG BUILD_TAGS
ENV CGO_ENABLED=0
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    BUILD_TAGS=${BUILD_TAGS} \
    make -f builder.Makefile test
