FROM golang:1.23.12-alpine3.21 AS builder

ARG GO_LDFLAGS

COPY . /go/src/github.com/kubeedge/kubeedge

RUN CGO_ENABLED=0 GO111MODULE=off go build -v -o /usr/local/bin/iptables-manager -ldflags "$GO_LDFLAGS -w -s" \
    github.com/kubeedge/kubeedge/cloud/cmd/iptablesmanager


FROM debian:12

COPY --from=builder /usr/local/bin/iptables-manager /usr/local/bin/iptables-manager

RUN apt-get update && apt-get -y install iptables

# Switch the iptables mode from the default nft mode back to the legacy mode, refer to https://wiki.debian.org/iptables
RUN update-alternatives --set iptables /usr/sbin/iptables-legacy \
     && update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy

ENTRYPOINT ["iptables-manager"]
