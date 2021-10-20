# Build the proxy-server binary
FROM golang:1.14-alpine3.11 as builder

# Copy in the go src
WORKDIR /go/src/sigs.k8s.io/apiserver-network-proxy

COPY . /go/src/github.com/kubeedge/kubeedge

# Build
ARG ARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -a -ldflags '-extldflags "-static"' -o edgesite-server github.com/kubeedge/kubeedge/edgesite/cmd/edgesite-server

# Copy the loader into a thin image
FROM scratch
WORKDIR /
COPY --from=builder /go/src/sigs.k8s.io/apiserver-network-proxy/edgesite-server .
ENTRYPOINT ["/edgesite-server"]
