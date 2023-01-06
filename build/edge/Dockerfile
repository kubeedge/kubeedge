ARG BUILD_FROM=golang:1.17.13-alpine3.16
ARG RUN_FROM=docker:dind

FROM ${BUILD_FROM} AS builder

ARG GO_LDFLAGS
ARG QEMU_ARCH=x86_64

RUN apk update && apk add curl && \
curl -L -o /tmp/qemu-${QEMU_ARCH}-static.tar.gz https://github.com/multiarch/qemu-user-static/releases/download/v3.0.0/qemu-${QEMU_ARCH}-static.tar.gz && \
tar -xzf /tmp/qemu-${QEMU_ARCH}-static.tar.gz -C /usr/bin

COPY . /go/src/github.com/kubeedge/kubeedge

RUN apk --no-cache update && \
apk --no-cache upgrade && \
apk --no-cache add build-base linux-headers sqlite-dev binutils-gold && \
CGO_ENABLED=1 GO111MODULE=off go build -v -o /usr/local/bin/edgecore -ldflags="${GO_LDFLAGS} -w -s -extldflags -static" \
github.com/kubeedge/kubeedge/edge/cmd/edgecore

FROM ${RUN_FROM}

COPY --from=builder /usr/bin/qemu* /usr/bin/

COPY --from=builder /usr/local/bin/edgecore /usr/local/bin/edgecore

ENTRYPOINT ["edgecore"]
