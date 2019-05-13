
# make kubeedge
.PHONY: default kubeedge
kubeedge:
	go build -o kubeedge keadm.go

.PHONY: clean
clean:
	rm kubeedge

# .PHONY: cross_build
# cross_build:
# 	export GOARCH=arm
# 	export GOOS="linux"
# 	export GOARM=6
# 	export CGO_ENABLED=1
# 	export CC=arm-linux-gnueabi-gcc
# 	go build keadm.go -o kubeedge

