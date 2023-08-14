FROM golang:1.19.12-alpine3.17 AS builder

ARG GO_LDFLAGS

COPY . /go/src/github.com/kubeedge/kubeedge

RUN CGO_ENABLED=0 GO111MODULE=off go build -v -o /usr/local/bin/controllermanager -ldflags "$GO_LDFLAGS -w -s" \
    github.com/kubeedge/kubeedge/cloud/cmd/controllermanager


FROM alpine:3.17

COPY --from=builder /usr/local/bin/controllermanager /usr/local/bin/controllermanager

ENTRYPOINT ["controllermanager"]
