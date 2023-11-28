FROM golang:1.20.10-alpine3.18 AS builder

WORKDIR /build

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go


FROM ubuntu:18.04

RUN mkdir -p kubeedge

COPY --from=builder /build/main kubeedge/
COPY ./config.yaml kubeedge/

WORKDIR kubeedge
