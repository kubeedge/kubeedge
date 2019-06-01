IMAGE_TAG ?= $(shell git describe --tags)
QEMU_ARCH ?= x86_64
ARCH ?= amd64
RELEASE_KUBEEDGE_ARCH ?= "arm" "amd64"
RELEASE_EDGE_ARCH ?= "arm" "amd64"
RELEASE_KEADM_ARCH ?= "amd64"
RELEASE_OS ?= "linux"
SHASUM ?= shasum -a 256
TAR ?= tar -zcvf

.PHONY: default all 
# build all
all: cloud edge keadm

.PHONY: cloud
# build cloud
cloud:
	$(MAKE) -C cloud

.PHONY: edge
# build edge
edge:
	$(MAKE) -C edge

.PHONY: keadm
# build keadm
keadm: 
	$(MAKE) -C keadm

.PHONY: clean
# clean
clean: 
	$(MAKE) -C cloud clean
	$(MAKE) -C edge clean
	$(MAKE) -C keadm clean
	$(RM) -r ./build/tmp
	$(RM) *.tar.gz
	$(RM) *.checksum

.PHONY: edge_test
# unit tests
edge_test:
	$(MAKE) -C edge test

.PHONY: edge_verify
# verify
edge_verify:
	$(MAKE) -C edge verify

.PHONY: edge_integration_test
edge_integration_test:
	$(MAKE) -C edge integration_test

.PHONY: edge_cross_build
edge_cross_build:
	$(MAKE) -C edge cross_build

.PHONY:
cloud_cross_build:
	$(MAKE) -C cloud cross_build

.PHONY: edge_small_build
edge_small_build:
	$(MAKE) -C edge small_build

.PHONY: cloud_lint
cloud_lint:
	$(MAKE) -C cloud lint

.PHONY: e2e_test
e2e_test:
	bash tests/e2e/scripts/execute.sh

.PHONY: performance_test
performance_test:
	bash tests/performance/scripts/jenkins.sh

.PHONY: cloudimage
cloudimage:
	docker build -t kubeedge/edgecontroller:${IMAGE_TAG} -f build/cloud/Dockerfile .

.PHONY: edgeimage
edgeimage:
	mkdir -p ./build/edge/tmp
	$(RM) -r ./build/edge/tmp/*
	curl -L -o ./build/edge/tmp/qemu-${QEMU_ARCH}-static.tar.gz https://github.com/multiarch/qemu-user-static/releases/download/v3.0.0/qemu-${QEMU_ARCH}-static.tar.gz 
	tar -xzf ./build/edge/tmp/qemu-${QEMU_ARCH}-static.tar.gz -C ./build/edge/tmp 
	docker build -t kubeedge/edgecore:${IMAGE_TAG} \
	--build-arg BUILD_FROM=${ARCH}/golang:1.12-alpine3.9 \
	--build-arg RUN_FROM=${ARCH}/docker:dind \
	-f build/edge/Dockerfile .

%.checksum: %.tar.gz
	$(SHASUM) $< > $@

create_installer_binaries:
	$(foreach os, $(RELEASE_OS), \
		$(foreach arch, $(RELEASE_KEADM_ARCH), \
			mkdir -p ./build/tmp/keadm/$(os)/$(arch)/kubeedge/; \
			echo $(IMAGE_TAG) > ./build/tmp/keadm/$(os)/$(arch)/kubeedge/version; \
			export GOOS=$(os); export GOARCH=$(arch); \
			$(MAKE) keadm; \
			mv ./keadm/kubeedge ./build/tmp/keadm/$(os)/$(arch)/kubeedge; \
			$(TAR) keadm-$(IMAGE_TAG)-$(os)-$(arch).tar.gz -C ./build/tmp/keadm/$(os)/$(arch) kubeedge; \
			$(MAKE) keadm-$(IMAGE_TAG)-$(os)-$(arch).checksum; \
		)\
	)

create_edge_binaries:
	$(foreach os, $(RELEASE_OS), \
		$(foreach arch, $(RELEASE_EDGE_ARCH), \
			mkdir -p ./build/tmp/edge/$(os)/$(arch)/kubeedge/edge; \
			echo $(IMAGE_TAG) > ./build/tmp/edge/$(os)/$(arch)/kubeedge/version; \
			export GOOS=$(os); export GOARCH=$(arch); \
			$(MAKE) edge; \
			mv ./edge/edge_core ./build/tmp/edge/$(os)/$(arch)/kubeedge/edge; \
			cp -r ./edge/conf ./build/tmp/edge/$(os)/$(arch)/kubeedge/edge/; \
			$(TAR) edge-$(IMAGE_TAG)-$(os)-$(arch).tar.gz -C ./build/tmp/edge/$(os)/$(arch) kubeedge; \
			$(MAKE) edge-$(IMAGE_TAG)-$(os)-$(arch).checksum; \
		) \
	)

create_kubeedge_binaries:
	$(foreach os, $(RELEASE_OS), \
		$(foreach arch, $(RELEASE_KUBEEDGE_ARCH), \
			mkdir -p ./build/tmp/kubeedge/$(os)/$(arch)/kubeedge/cloud; \
			mkdir -p ./build/tmp/kubeedge/$(os)/$(arch)/kubeedge/edge; \
			echo $(IMAGE_TAG) > ./build/tmp/kubeedge/$(os)/$(arch)/kubeedge/version; \
			export GOOS=$(os); export GOARCH=$(arch); \
			$(MAKE) edge; \
			$(MAKE) cloud; \
			mv ./edge/edge_core ./build/tmp/kubeedge/$(os)/$(arch)/kubeedge/edge; \
			cp -r ./edge/conf ./build/tmp/kubeedge/$(os)/$(arch)/kubeedge/edge/; \
			mv ./cloud/edgecontroller ./build/tmp/kubeedge/$(os)/$(arch)/kubeedge/cloud; \
			cp -r ./cloud/conf ./build/tmp/kubeedge/$(os)/$(arch)/kubeedge/cloud/; \
			$(TAR) kubeedge-$(IMAGE_TAG)-$(os)-$(arch).tar.gz -C ./build/tmp/kubeedge/$(os)/$(arch) kubeedge; \
			$(MAKE) kubeedge-$(IMAGE_TAG)-$(os)-$(arch).checksum; \
		)\
	)

.PHONY: release
# create binaries
release: create_kubeedge_binaries create_edge_binaries create_installer_binaries
