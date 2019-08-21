
GO_LDFLAGS=-ldflags '$(shell ../hack/lib/version.sh)'
# make edgecore
.PHONY: default edgecore
edgecore:
	go build ${GO_LDFLAGS}  cmd/edgecore/edgecore.go

# unit tests
.PHONY: test
ifeq ($(WHAT),)
       TEST_DIR="./pkg/"
else
       TEST_DIR=${WHAT}	
endif

export GOARCHAIUS_CONFIG_PATH=$(CURDIR)
test:
	find ${TEST_DIR} -name "*_test.go"| xargs -I{} dirname {} | uniq | xargs -I{} go test ${T} {}

# lint
.PHONY: lint
lint:
	golangci-lint run --disable-all -E gofmt -E golint -E misspell --deadline '10m' ./...
	go vet ./...

.PHONY: integration_test
integration_test:
	bash test/integration/scripts/execute.sh

.PHONY: clean
clean:
	bash -x hack/clean.sh

.PHONY: cross_build
cross_build:
	export GOARCH=arm \
        export GOOS="linux" \
        export GOARM=6 \
        export CGO_ENABLED=1 \
        export CC=arm-linux-gnueabi-gcc; \
        go build cmd/edgecore/edgecore.go

.PHONY: armv7
armv7:
	export GOARCH=arm \
        export GOOS="linux" \
        export GOARM=7 \
        export CGO_ENABLED=1 \
        export CC=arm-linux-gnueabi-gcc; \
        go build cmd/edgecore/edgecore.go

.PHONY: armv8
armv8:
	export GOARCH=arm64 \
        export GOOS="linux" \
        export CGO_ENABLED=1 \
        export CC=aarch64-linux-gnu-gcc; \
        go build cmd/edgecore/edgecore.go

.PHONY: small_build
small_build:
	go build -ldflags="-w -s -extldflags -static" cmd/edgecore/edgecore.go
	# There are options in upx-ucl like --best, which has best compression,
	# but also has a drawback of taking too long for big files, we are using -9 since it takes less time
	# Pls use --help for more information
	upx-ucl -9 edgecore
