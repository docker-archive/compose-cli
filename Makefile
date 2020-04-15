# Copyright (c) 2020 Docker Inc.

# Permission is hereby granted, free of charge, to any person
# obtaining a copy of this software and associated documentation
# files (the "Software"), to deal in the Software without
# restriction, including without limitation the rights to use, copy,
# modify, merge, publish, distribute, sublicense, and/or sell copies
# of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED,
# INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
# IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
# HOLDERS BE LIABLE FOR ANY CLAIM,
# DAMAGES OR OTHER LIABILITY,
# WHETHER IN AN ACTION OF CONTRACT,
# TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH
# THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

PACKAGES=$(shell go list ./... | grep -v /vendor/)
GIT_COMMIT=$(shell git rev-parse --short HEAD)
GOOS ?= $(shell go env GOOS)

export GO111MODULE=off

all: protos example-aci example-d2 cli

cli:
	cd cmd && go build -v -o ../bin/docker

protos:
	@protobuild --quiet ${PACKAGES}

example-aci:
	cd example/backend && go build -v -o ../../bin/backend-example-azure_aci

example-d2:
	cd example2/backend && go build -v -o ../../bin/backend-example-d2

FORCE:

.PHONY: protos example cli
