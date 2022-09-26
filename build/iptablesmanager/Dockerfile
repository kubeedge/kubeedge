FROM golang:1.16-alpine3.13 AS builder

ARG GO_LDFLAGS

COPY . /go/src/github.com/kubeedge/kubeedge

RUN CGO_ENABLED=0 GO111MODULE=off go build -v -o /usr/local/bin/iptables-manager -ldflags "$GO_LDFLAGS -w -s" \
    github.com/kubeedge/kubeedge/cloud/cmd/iptablesmanager


FROM alpine:3.13

COPY --from=builder /usr/local/bin/iptables-manager /usr/local/bin/iptables-manager

RUN apk add --update-cache \
    iptables \ 
    && rm -rf /var/cache/apk/* \
    && apk --no-cache add dpkg

# Add support for auto-detection of iptables mode
COPY --from=builder /go/src/github.com/kubeedge/kubeedge/build/iptablesmanager/iptables-wrapper /sbin/iptables-wrapper
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-legacy 50
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-nft 50
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-wrapper 100 \
    --slave /sbin/iptables-restore iptables-restore /sbin/iptables-wrapper \
    --slave /sbin/iptables-save iptables-save /sbin/iptables-wrapper

ENTRYPOINT ["iptables-manager"]
