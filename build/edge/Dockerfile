FROM golang:1.12.1-alpine3.9 AS builder

COPY . /go/src/github.com/kubeedge/kubeedge

RUN apk --no-cache update && \
apk --no-cache upgrade && \
apk --no-cache add build-base linux-headers sqlite-dev && \
CGO_ENABLED=1 go build -v -o /usr/local/bin/edge_core -ldflags="-w -s -extldflags -static" \
github.com/kubeedge/kubeedge/edge/cmd


FROM alpine:3.9

ENV GOARCHAIUS_CONFIG_PATH /etc/kubeedge/edge
ENV mqtt.server tcp://127.0.0.1:1883
ENV mqtt.mode 2
ENV edgehub.websocket.url wss://127.0.0.1:10000/e632aba927ea4ac2b575ec1603d56f10/fb4ebb70-2783-42b8-b3ef-63e2fd6d242e/events
ENV edgehub.websocket.certfile /etc/kubeedge/edge/certs/edge.crt
ENV edgehub.websocket.keyfile /etc/kubeedge/edge/certs/edge.key
ENV database.source /var/lib/kubeedge/edge.db

VOLUME ["/etc/kubeedge/edge/certs", "/var/lib/edged", "/var/lib/kubeedge", "/var/run/docker.sock"]

COPY --from=builder /usr/local/bin/edge_core /usr/local/bin/edge_core
COPY --from=builder /go/src/github.com/kubeedge/kubeedge/edge/conf /etc/kubeedge/edge/conf

ENTRYPOINT ["edge_core"]
