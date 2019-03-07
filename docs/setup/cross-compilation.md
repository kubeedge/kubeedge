# Cross Compiling KubeEdge 

## For ARM Architecture from x86 Architecture 

Clone KubeEdge

```shell
# Build and run KubeEdge on a ARMv6 target device.

git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
sudo apt-get install gcc-arm-linux-gnueabi
export GOARCH=arm
export GOOS="linux"
export GOARM=6                             #Pls give the appropriate arm version of your device  
export CGO_ENABLED=1
export CC=arm-linux-gnueabi-gcc
make # or `make edge_core`
```
