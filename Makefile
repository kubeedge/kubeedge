# make all builds both cloud and edge binaries
.PHONY: all  
ifeq ($(WHAT),)
all:
	cd cloud && $(MAKE)
	cd edge && $(MAKE)
	cd keadm && $(MAKE)
	cd edgesite && $(MAKE)
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

.PHONY: cloud_test
cloud_test:
	$(MAKE) -C cloud test

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
#	bash tests/e2e/scripts/execute.sh device_crd
#	This has been commented temporarily since there is an issue of CI using same master for all PRs, which is causing failures when run parallely
	bash tests/e2e/scripts/execute.sh

.PHONY: performance_test
performance_test:
	bash tests/performance/scripts/jenkins.sh

IMAGE_TAG ?= $(shell git describe --tags)

.PHONY: cloudimage
cloudimage:
	docker build -t kubeedge/edgecontroller:${IMAGE_TAG} -f build/cloud/Dockerfile .

.PHONY: keadm_lint
keadm_lint:
	make -C keadm lint

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

.PHONY: depcheck
depcheck:
	dep check

.PHONY: bluetoothdevice
bluetoothdevice:
	make -C device/bluetooth_mapper

.PHONY: bluetoothdevice_image
	make -C device/bluetooth_mapper_docker

.PHONY: bluetoothdevice_lint
bluetoothdevice_lint:
	make -C device/bluetooth_mapper lint
