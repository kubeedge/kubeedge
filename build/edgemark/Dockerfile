ARG BUILD_FROM=golang:1.17.13-alpine3.16

FROM ${BUILD_FROM} AS builder

ARG GO_LDFLAGS

COPY . /go/src/github.com/kubeedge/kubeedge

RUN apk --no-cache update && \
apk --no-cache upgrade && \
apk --no-cache add build-base linux-headers sqlite-dev binutils-gold && \
CGO_ENABLED=1 GO111MODULE=off go build -v -o /usr/local/bin/edgemark -ldflags="${GO_LDFLAGS} -w -s -extldflags -static" \
github.com/kubeedge/kubeedge/edge/cmd/edgemark

FROM alpine:3.16

COPY --from=builder /usr/local/bin/edgemark /usr/local/bin/edgemark

ENTRYPOINT ["edgemark"]