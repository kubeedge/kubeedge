FROM golang:1.14-alpine3.11 AS builder

ARG GO_LDFLAGS

COPY . /go/src/github.com/kubeedge/kubeedge

RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/csidriver -ldflags="${GO_LDFLAGS} -w -s" \
github.com/kubeedge/kubeedge/cloud/cmd/csidriver

FROM alpine:3.11

COPY --from=builder /usr/local/bin/csidriver /usr/local/bin/csidriver

ENTRYPOINT ["csidriver"]
