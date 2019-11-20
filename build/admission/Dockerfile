FROM golang:1.12-alpine3.10 AS builder

COPY . /go/src/github.com/kubeedge/kubeedge

RUN CGO_ENABLED=0 go build -v -o /usr/local/bin/admission -ldflags="-w -s" \
github.com/kubeedge/kubeedge/cloud/cmd/admission


FROM alpine:3.9

COPY --from=builder /usr/local/bin/admission /usr/local/bin/admission

ENTRYPOINT ["admission"]
