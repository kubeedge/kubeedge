# Build from source

If you want to build KubeEdge from source you would need a working installation of the Go 1.14+ [toolchain](https://github.com/golang/tools) (`GOPATH`, `PATH=${GOPATH}/bin:${PATH}`).

Clone repo:
```bash
$ git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
```

Then go to KubeEdge git repo and build as following:

```bash
$ cd $GOPATH/src/github.com/kubeedge/kubeedge
$ make build WHAT=keadm
```

In most of the cases, when you are trying to compile KubeEdge edgecore on Raspberry Pi or any other device, you may run out of memory, in that case, it is advisable to cross-compile the Edgecore binary and transfer it to your edge device.

## Cross build

If you want to build binaries for other arch different from your machine's, for example, build `keadm` for arm32 on x86 machine:

```bash
# install gcc-arm-linux-gnueabi with your OS package manager
$ make crossbuild WHAT=keadm GOARM=GOARM7
```

for arm64:

```bash
# install gcc-aarch64-linux-gnu with your OS package manager
$ make crossbuild WHAT=keadm GOARM=GOARM8
```
