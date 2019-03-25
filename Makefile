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

.PHONY: edge_small_build
edge_small_build:
	cd edge && $(MAKE) small_build

.PHONY: edgecontroller
edgecontroller:
	cd cloud/edgecontroller && $(MAKE)

.PHONY: e2e_test
e2e_test:
	bash tests/e2e/scripts/execute.sh

IMAGE_TAG ?= $(shell git describe --tags)

.PHONY: cloudimage
cloudimage:
	docker build -t kubeedge/edgecontroller:${IMAGE_TAG} -f build/cloud/Dockerfile .
