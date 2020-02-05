.PHONY: small_build
small_build: verify
	go build -ldflags="-w -s -extldflags -static" cmd/edgesite/edgesite.go
	# There are options in upx-ucl like --best, which has best compression,
	# but also has a drawback of taking too long for big files, we are using -9 since it takes less time
	# Pls use --help for more information
	upx-ucl -9 edgesite
