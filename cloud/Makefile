# make cloud part

# get ldflags 
GO_LDFLAGS=-ldflags '$(shell ../hack/lib/version.sh)'

.PHONY: default cloudcore admission

default: cloudcore admission

cloudcore:
	go build ${GO_LDFLAGS} cmd/cloudcore/cloudcore.go

admission:
	go build ${GO_LDFLAGS} cmd/admission/admission.go

.PHONY: lint
lint:
	golangci-lint run --disable-all -E gofmt -E golint --deadline '10m' ./...
	go vet ./...

# Build and run tests.
#
# Args:
#   WHAT: Directory names to test.  All *_test.go files under these
#     directories will be run.  If not specified, "everything" will be tested.
#   KUBE_TEST_ARGS: Test case name to test.
#   GOFLAGS: go flags to be passed like verbose mode etc.
# Example:
#   make test
#   make test GOFLAGS=-v
#   make test WHAT=pkg/controller/manager/ GOFLAGS=-v
#   make test GOFLAGS="-v" KUBE_TEST_ARGS="-run ^TestEventToMessage"
.PHONY: test
test:
	hack/test.sh $(WHAT)

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

