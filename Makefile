# make all builds both cloud and edge binaries
.PHONY: all  
ifeq ($(WHAT),)
all:
	cd cloud && $(MAKE)
	cd edge && $(MAKE)
	cd keadm && $(MAKE)
	cd edgesite && $(MAKE)
else ifeq ($(WHAT),cloudcore)
# make all WHAT=cloudcore
all:
	cd cloud && $(MAKE) cloudcore
else ifeq ($(WHAT),admission)
# make all WHAT=admission
all:
	cd cloud && $(MAKE) admission
else ifeq ($(WHAT),edgecore)
all:
# make all WHAT=edgecore
	cd edge && $(MAKE)
else ifeq ($(WHAT),edgesite)
all:
# make all WHAT=edgesite
	$(MAKE) -C edgesite
else ifeq ($(WHAT),keadm)
all:
# make all WHAT=keadm
	cd keadm && $(MAKE)
else
# invalid entry
all:
	@echo $S"invalid option please choose to build either cloudcore, admission, edgecore, keadm, edgesite or all together"
endif

# unit tests
.PHONY: edge_test
edge_test:
	cd edge && $(MAKE) test

.PHONY: cloud_test
cloud_test:
	$(MAKE) -C cloud test

# lint
.PHONY: edge_lint
edge_lint:
	cd edge && $(MAKE) lint

.PHONY: edge_integration_test
edge_integration_test:
	cd edge && $(MAKE) integration_test

.PHONY: edge_cross_build
edge_cross_build:
	cd edge && $(MAKE) cross_build

.PHONY: edge_cross_build_v7
edge_cross_build_v7:
	$(MAKE) -C edge armv7

.PHONY: edge_cross_build_v8
edge_cross_build_v8:
	$(MAKE) -C edge armv8

.PHONY: edgesite_cross_build
edgesite_cross_build:
	$(MAKE) -C edgesite cross_build

.PHONY: edge_small_build
edge_small_build:
	cd edge && $(MAKE) small_build

.PHONY: edgesite_cross_build_v7
edgesite_cross_build_v7:
	$(MAKE) -C edgesite armv7

.PHONY: edgesite_cross_build_v8
edgesite_cross_build_v8:
	$(MAKE) -C edgesite armv8

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

.PHONY: keadm_lint
keadm_lint:
	make -C keadm lint

QEMU_ARCH ?= x86_64
ARCH ?= amd64

IMAGE_TAG ?= $(shell git describe --tags)

.PHONY: cloudimage
cloudimage:
	docker build -t kubeedge/cloudcore:${IMAGE_TAG} -f build/cloud/Dockerfile .

.PHONY: admissionimage
admissionimage:
	docker build -t kubeedge/admission:${IMAGE_TAG} -f build/admission/Dockerfile .

.PHONY: csidriverimage
csidriverimage:
	docker build -t kubeedge/csidriver:${IMAGE_TAG} -f build/csidriver/Dockerfile .

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

.PHONY: edgesiteimage
edgesiteimage:
	mkdir -p ./build/edgesite/tmp
	rm -rf ./build/edgesite/tmp/*
	curl -L -o ./build/edgesite/tmp/qemu-${QEMU_ARCH}-static.tar.gz https://github.com/multiarch/qemu-user-static/releases/download/v3.0.0/qemu-${QEMU_ARCH}-static.tar.gz
	tar -xzf ./build/edgesite/tmp/qemu-${QEMU_ARCH}-static.tar.gz -C ./build/edgesite/tmp
	docker build -t kubeedge/edgesite:${IMAGE_TAG} \
	--build-arg BUILD_FROM=${ARCH}/golang:1.12-alpine3.9 \
	--build-arg RUN_FROM=${ARCH}/docker:dind \
	-f build/edgesite/Dockerfile .

.PHONY: vendorCheck
vendorCheck:
	bash build/tools/verifyVendor.sh

.PHONY: bluetoothdevice
bluetoothdevice:
	make -C mappers/bluetooth_mapper

.PHONY: bluetoothdevice_image
	make -C mappers/bluetooth_mapper_docker

.PHONY: bluetoothdevice_lint
bluetoothdevice_lint:
	make -C mappers/bluetooth_mapper lint
