# Build

## Build using Golang


```shell
$ go get github.com/kubeedge/kubeedge/cloud/cmd/cloudcore
$ go get github.com/kubeedge/kubeedge/cloud/cmd/admission
$ go get github.com/kubeedge/kubeedge/edge/cmd/edgecore
$ go get github.com/kubeedge/kubeedge/edgesite/cmd/edgesite
$ go get github.com/kubeedge/kubeedge/keadm/cmd/keadm
```

All binaries would locate at `$GOPATH/bin`.

## Build through Make

You can build through `make all` after download the repository:

```shell
$ make all HELP=y
# Build code.
#
# Args:
#   WHAT: binary names to build. support: cloudcore admission edgecore edgesite keadm
#         the build will produce executable files under _output
#         If not specified, "everything" will be built.
#
# Example:
#   make
#   make all
#   make all HELP=y
#   make all WHAT=cloudcore

```

