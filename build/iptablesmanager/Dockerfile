FROM golang:1.17.13-alpine3.16 AS builder

ARG GO_LDFLAGS

COPY . /go/src/github.com/kubeedge/kubeedge

RUN CGO_ENABLED=0 GO111MODULE=off go build -v -o /usr/local/bin/iptables-manager -ldflags "$GO_LDFLAGS -w -s" \
    github.com/kubeedge/kubeedge/cloud/cmd/iptablesmanager


FROM alpine:3.16

COPY --from=builder /usr/local/bin/iptables-manager /usr/local/bin/iptables-manager

RUN apk add --update-cache \
    iptables \
    && rm -rf /var/cache/apk/*

ENTRYPOINT ["iptables-manager"]
