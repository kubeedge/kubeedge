# make all builds both cloud and edge binaries

PKG=github.com/kubeedge/kubeedge

# Used to populate variables in version package.
VERSION=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always)
REVISION=$(shell git rev-parse HEAD)$(shell if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)
GOVERSION=$(shell go version |awk -F ' ' '{printf $$3}')
CURRENT_BRANCH=$(shell git symbolic-ref --short -q HEAD)

export GO_LDFLAGS=-ldflags '-s -w -X $(PKG)/version.Version=$(VERSION) -X $(PKG)/version.Revision=$(REVISION) -X $(PKG)/version.Package=$(PKG) -X $(PKG)/version.GoVersion=$(GOVERSION) -X $(PKG)/version.Branch=$(CURRENT_BRANCH) $(EXTRA_LDFLAGS)'

#Replaces ":" (*nix), ";" (windows) with newline for easy parsing
GOPATHS=$(shell echo ${GOPATH} | tr ":" "\n" | tr ";" "\n")
TESTFLAGS_RACE= -race

export GO_BUILD_FLAGS=
# See Golang issue re: '-trimpath': https://github.com/golang/go/issues/13809
export GO_GCFLAGS=$(shell				\
	set -- ${GOPATHS};			\
	echo "-gcflags=-trimpath=$${1}/src";	\
	)

# Flags passed to `go test`
#export TESTFLAGS ?= -v $(TESTFLAGS_RACE)

.PHONY: all
ifeq ($(WHAT),)
all:
	cd cloud && $(MAKE)
	cd edge && $(MAKE)
	cd keadm && $(MAKE)
else ifeq ($(WHAT),cloud)
# make all what=cloud, build cloud binary
all:
	cd cloud && $(MAKE)
else ifeq ($(WHAT),edge)
all:
# make all what=edge, build edge binary
	cd edge && $(MAKE)
else ifeq ($(WHAT),keadm)
all:
# make all what=edge, build edge binary
	cd keadm && $(MAKE)
else
# invalid entry
all:
	@echo $S"invalid option please choose to build either cloud, edge or both"
endif

# unit tests
.PHONY: edge_test
edge_test:
	cd edge && $(MAKE) test

# verify
.PHONY: edge_verify
edge_verify:
	cd edge && $(MAKE) verify

.PHONY: edge_integration_test
edge_integration_test:
	cd edge && $(MAKE) integration_test

.PHONY: edge_cross_build
edge_cross_build:
	cd edge && $(MAKE) cross_build

.PHONY: edge_small_build
edge_small_build:
	cd edge && $(MAKE) small_build

.PHONY: cloud_lint
cloud_lint:
	cd cloud && $(MAKE) lint

.PHONY: e2e_test
e2e_test:
	bash tests/e2e/scripts/execute.sh

.PHONY: performance_test
performance_test:
	bash tests/performance/scripts/jenkins.sh

IMAGE_TAG ?= $(shell git describe --tags)

.PHONY: cloudimage
cloudimage:
	docker build -t kubeedge/edgecontroller:${IMAGE_TAG} -f build/cloud/Dockerfile .

QEMU_ARCH ?= x86_64
ARCH ?= amd64

.PHONY: edgeimage
edgeimage:
	mkdir -p ./build/edge/tmp
	rm -rf ./build/edge/tmp/*
	curl -L -o ./build/edge/tmp/qemu-${QEMU_ARCH}-static.tar.gz https://github.com/multiarch/qemu-user-static/releases/download/v3.0.0/qemu-${QEMU_ARCH}-static.tar.gz 
	tar -xzf ./build/edge/tmp/qemu-${QEMU_ARCH}-static.tar.gz -C ./build/edge/tmp 
	docker build -t kubeedge/edgecore:${IMAGE_TAG} \
	--build-arg BUILD_FROM=${ARCH}/golang:1.12-alpine3.9 \
	--build-arg RUN_FROM=${ARCH}/docker:dind \
	-f build/edge/Dockerfile .
  
