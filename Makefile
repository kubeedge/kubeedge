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

