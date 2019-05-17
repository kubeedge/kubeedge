
# make edge_site
.PHONY: default edge_site
edge_site:
	go build cmd/edgesite.go

.PHONY: cross_build
cross_build:
	export GOARCH=arm \
        export GOOS="linux" \
        export GOARM=6 \
        export CGO_ENABLED=1 \
        export CC=arm-linux-gnueabi-gcc; \
        go build cmd/edgesite.go

.PHONY: small_build
small_build:
	go build -ldflags="-w -s -extldflags -static" cmd/edgesite.go
	upx-ucl --best edge_site
