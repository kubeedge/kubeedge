lint:
	golangci-lint run --deadline '10m' --disable-all -E golint ./...
	go vet ./...

.PHONY: clean
clean:
	$(RM) keadm

