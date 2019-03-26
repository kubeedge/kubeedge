FROM golang:1.12.1-alpine3.9 AS builder

COPY . /go/src/github.com/kubeedge/kubeedge

RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/edgecontroller -ldflags="-w -s" \
github.com/kubeedge/kubeedge/cloud/edgecontroller/cmd


FROM alpine:3.9

ENV GOARCHAIUS_CONFIG_PATH /etc/kubeedge/cloud

VOLUME ["/etc/kubeedge/cloud/certs", "/etc/kubeedge/cloud/conf"]

COPY --from=builder /usr/local/bin/edgecontroller /usr/local/bin/edgecontroller

ENTRYPOINT ["edgecontroller"]
