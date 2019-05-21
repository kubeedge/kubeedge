
# make edge_core
.PHONY: default edge_core
edge_core:
	go build cmd/edge_core.go

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

# verify
.PHONY: verify
verify:
	bash -x hack/verify.sh

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
        go build cmd/edge_core.go

.PHONY: small_build
small_build:
	go build -ldflags="-w -s -extldflags -static" cmd/edge_core.go
	upx-ucl --best edge_core
