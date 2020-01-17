# make cloud part

# get ldflags 
GO_LDFLAGS=-ldflags '$(shell ../hack/lib/version.sh)'
CLOUD_CORE_BINARY=cloudcore
ADMISSION_BINARY=admission
BINDIR=/usr/local/bin
.PHONY: default cloudcore admission

default: cloudcore admission

cloudcore: verify clean
	go build ${GO_LDFLAGS} -o ${CLOUD_CORE_BINARY} cmd/cloudcore/cloudcore.go

admission: verify clean
	go build ${GO_LDFLAGS} -o ${ADMISSION_BINARY} cmd/admission/admission.go

.PHONY: clean
clean:
	rm -f ${CLOUD_CORE_BINARY}
	rm -f ${ADMISSION_BINARY}

.PHONY: lint
lint:
	golangci-lint run --skip-dirs 'pkg/client' --disable-all -E gofmt -E golint --deadline '10m' ./...
	go vet ./...

.PHONY: verify 
verify:
	bash ../hack/verify-golang.sh

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

.PHONY: install
install: default
	install ${CLOUD_CORE_BINARY} ${BINDIR}
	install ${ADMISSION_BINARY} ${BINDIR}

.PHONY: uninstall
uninstall:
	rm -f ${BINDIR}/${CLOUD_CORE_BINARY}
	rm -f ${BINDIR}/${ADMISSION_BINARY}
