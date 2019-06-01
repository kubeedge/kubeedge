
# make edgecontroller
.PHONY: default edgecontroller clean
edgecontroller:
	go build cmd/edgecontroller.go

clean:
	$(RM) edgecontroller

lint:
	golangci-lint run --disable-all -E gofmt -E golint ./...
	go vet ./...

generate-code:
	./hack/update-codegen.sh

