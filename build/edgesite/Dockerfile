ARG BUILD_FROM=golang:1.13.8-alpine3.10
ARG RUN_FROM=docker:dind

FROM ${BUILD_FROM} AS builder

ARG GO_LDFLAGS
COPY . /go/src/github.com/kubeedge/kubeedge

RUN apk --no-cache update && \
apk --no-cache upgrade && \
apk --no-cache add build-base linux-headers sqlite-dev && \
CGO_ENABLED=1 go build -v -o /usr/local/bin/edgesite -ldflags="${GO_LDFLAGS} -w -s -extldflags -static" \
github.com/kubeedge/kubeedge/edgesite/cmd/edgesite

FROM ${RUN_FROM}

ENV GOARCHAIUS_CONFIG_PATH /etc/kubeedge/edgesite
ENV database.source /var/lib/kubeedge/edge.db

COPY --from=builder /usr/local/bin/edgesite /usr/local/bin/edgesite

ENTRYPOINT ["edgesite"]
