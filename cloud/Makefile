# make cloud part


.PHONY: 

default: 


# Build and run integration tests.
#
# Args:
#   WHAT: Directory names to test.  All *_test.go files under these
#     directories will be run.  If not specified, "everything" will be tested.
#   KUBE_TEST_ARGS: Test case name to test.
#   GOFLAGS: go flags to be passed like verbose mode etc.
# Example:
#   make test-integration
#   make test-integration GOFLAGS=-v
#   make test-integration GOFLAGS=-v KUBE_TEST_ARGS="-run ^TestInvalidDeviceModel$"
#   make test-integration GOFLAGS=-v WHAT=./test/integration/crds/
.PHONY: test-integration
test-integration:
	hack/test-integration.sh

.PHONY: generate-code
generate-code:
	./hack/update-codegen.sh

