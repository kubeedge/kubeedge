FROM golang:1.12-alpine3.9 AS builder

COPY . /go/src/github.com/kubeedge/kubeedge

RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/cloudcore -ldflags="-w -s" \
github.com/kubeedge/kubeedge/cloud/cmd/cloudcore


FROM gcr.io/distroless/static

ENV GOARCHAIUS_CONFIG_PATH /etc/kubeedge/cloud

VOLUME ["/etc/kubeedge/certs", "/etc/kubeedge/cloud/conf"]

COPY --from=builder /usr/local/bin/cloudcore /usr/local/bin/cloudcore

ENTRYPOINT ["cloudcore"]
