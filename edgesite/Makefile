GO_LDFLAGS=-ldflags '$(shell ../hack/lib/version.sh)'
# make edge_site
.PHONY: default edge_site
edge_site:
	go build ${GO_LDFLAGS} cmd/edgesite.go

.PHONY: cross_build
cross_build:
	export GOARCH=arm \
        export GOOS="linux" \
        export GOARM=6 \
        export CGO_ENABLED=1 \
        export CC=arm-linux-gnueabi-gcc; \
        go build cmd/edgesite.go

.PHONY: armv7
armv7:
	export GOARCH=arm \
        export GOOS="linux" \
        export GOARM=7 \
        export CGO_ENABLED=1 \
        export CC=arm-linux-gnueabi-gcc; \
        go build cmd/edgesite.go

.PHONY: armv8
armv8:
	export GOARCH=arm64 \
        export GOOS="linux" \
        export CGO_ENABLED=1 \
        export CC=aarch64-linux-gnu-gcc; \
        go build cmd/edgesite.go

.PHONY: small_build
small_build:
	go build -ldflags="-w -s -extldflags -static" cmd/edgesite.go
	upx-ucl --best edge_site
