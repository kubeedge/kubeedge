# Build

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
