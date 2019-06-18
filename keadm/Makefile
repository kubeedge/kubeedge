
# make keadm
.PHONY: default keadm
keadm:
	go build -o keadm keadm.go

lint:
	golangci-lint run --disable-all -E golint ./...
	go vet ./...

.PHONY: clean
clean:
	$(RM) keadm

# .PHONY: cross_build
# cross_build:
# 	export GOARCH=arm
# 	export GOOS="linux"
# 	export GOARM=6
# 	export CGO_ENABLED=1
# 	export CC=arm-linux-gnueabi-gcc
# 	go build keadm.go -o keadm

