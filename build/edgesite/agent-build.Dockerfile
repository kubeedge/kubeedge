# Build the proxy-agent binary
FROM golang:1.22.9-alpine3.19 as builder

WORKDIR /go/src/sigs.k8s.io/apiserver-network-proxy
COPY . /go/src/github.com/kubeedge/kubeedge

# Build
ARG ARCH
RUN CGO_ENABLED=0 GO111MODULE=off GOOS=linux GOARCH=${ARCH} go build -a -ldflags '-extldflags "-static"' -o edgesite-agent github.com/kubeedge/kubeedge/edgesite/cmd/edgesite-agent

# Copy the loader into a thin image
FROM scratch
WORKDIR /
COPY --from=builder /go/src/sigs.k8s.io/apiserver-network-proxy/edgesite-agent .
ENTRYPOINT ["/edgesite-agent"]
