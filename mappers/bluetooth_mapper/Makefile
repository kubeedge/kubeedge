
# make bluetooth_mapper
.PHONY: default bluetooth_mapper lint bluetooth_mapper_image
bluetooth_mapper:
	go build main.go

lint:
	golangci-lint run --disable-all -E golint --deadline '10m' ./...
	go vet ./...

bluetooth_mapper_image: bluetooth_mapper
	docker build -t bluetooth_mapper:v1.0 .
