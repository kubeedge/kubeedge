# make edge_core
.PHONY: edge_core
edge_core:
	cd edge && $(MAKE)

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

.PHONY: edgecontroller
edgecontroller:
	cd cloud/edgecontroller && $(MAKE)

IMAGE_TAG ?= $(shell git describe --tags)

.PHONY: cloudimage
cloudimage:
	docker build -t kubeedge/edgecontroller:${IMAGE_TAG} -f build/cloud/Dockerfile .

.PHONY: certgenimage
certgenimage:
	docker build -t kubeedge/certgen:${IMAGE_TAG} -f build/tools/Dockerfile build/tools

.PHONY: edgeimage
edgeimage:
	docker build -t kubeedge/edgecore:${IMAGE_TAG} -f build/edge/Dockerfile .
