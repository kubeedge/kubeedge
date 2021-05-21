# Build the proxy-server binary
FROM golang:1.13.4 as builder

# Copy in the go src
WORKDIR /go/src/sigs.k8s.io/apiserver-network-proxy

COPY . /go/src/github.com/kubeedge/kubeedge

# Build
ARG ARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -a -ldflags '-extldflags "-static"' -o proxy-server github.com/kubeedge/kubeedge/edgesite/cmd/server

# Copy the loader into a thin image
FROM scratch
WORKDIR /
COPY --from=builder /go/src/sigs.k8s.io/apiserver-network-proxy/proxy-server .
ENTRYPOINT ["/proxy-server"]
