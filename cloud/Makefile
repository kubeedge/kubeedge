
# make edgecontroller
.PHONY: default edgecontroller
edgecontroller:
	go build cmd/edgecontroller.go
lint:
	golangci-lint run --disable-all -E gofmt -E golint ./...
	go vet ./...

generate-code:
	./hack/update-codegen.sh

