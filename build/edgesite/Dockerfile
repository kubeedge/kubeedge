ARG BUILD_FROM=golang:1.14-alpine3.11
ARG RUN_FROM=docker:dind

FROM ${BUILD_FROM} AS builder

ARG GO_LDFLAGS
ARG QEMU_ARCH=x86_64
COPY ./build/edgesite/tmp/qemu-${QEMU_ARCH}-static /usr/bin/
COPY . /go/src/github.com/kubeedge/kubeedge

RUN apk --no-cache update && \
apk --no-cache upgrade && \
apk --no-cache add build-base linux-headers sqlite-dev && \
CGO_ENABLED=1 go build -v -o /usr/local/bin/edgesite -ldflags="${GO_LDFLAGS} -w -s -extldflags -static" \
github.com/kubeedge/kubeedge/edgesite/cmd/edgesite

FROM ${RUN_FROM}

COPY --from=builder /usr/bin/qemu* /usr/bin/

ENV GOARCHAIUS_CONFIG_PATH /etc/kubeedge/edgesite
ENV database.source /var/lib/kubeedge/edge.db

COPY --from=builder /usr/local/bin/edgesite /usr/local/bin/edgesite

ENTRYPOINT ["edgesite"]
