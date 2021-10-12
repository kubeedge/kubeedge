# Build the proxy-agent binary
FROM golang:1.14-alpine3.11 as builder

WORKDIR /go/src/sigs.k8s.io/apiserver-network-proxy
COPY . /go/src/github.com/kubeedge/kubeedge

# Build
ARG ARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -a -ldflags '-extldflags "-static"' -o edgexmanager github.com/kubeedge/kubeedge/edgexmanager/main

# Copy the loader into a thin image
FROM scratch
WORKDIR /
COPY --from=builder /go/src/github.com/kubeedge/kubeedge/edgexmanager .
ENTRYPOINT ["/edgexmanager"]

