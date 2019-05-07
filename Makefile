# make all builds both cloud and edge binaries
.PHONY: all  
ifeq ($(WHAT),)
all:
	cd cloud && $(MAKE)
	cd edge && $(MAKE)
else ifeq ($(WHAT),cloud)
# make all what=cloud, build cloud binary
all:
	cd cloud && $(MAKE)
else ifeq ($(WHAT),edge)
all:
# make all what=edge, build edge binary
	cd edge && $(MAKE)
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

.PHONY: edgeimage
edgeimage:
	docker build -t kubeedge/edgecore:${IMAGE_TAG} -f build/edge/Dockerfile .
