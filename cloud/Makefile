# make edgecontroller
TEST_DIR ?= "./"
.PHONY: default edgecontroller test
edgecontroller:
	go build cmd/edgecontroller.go
lint:
	golangci-lint run --disable-all -E gofmt -E golint ./...
	go vet ./...

test:
	find ${TEST_DIR} -name "*_test.go"| xargs -I{} dirname {} | uniq | xargs -I{} go test ${T} {}

generate-code:
	./hack/update-codegen.sh

