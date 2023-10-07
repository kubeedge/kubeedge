FROM ubuntu:16.04

RUN mkdir -p kubeedge

COPY ./cmd/main kubeedge/
COPY ./config.yaml kubeedge/

WORKDIR kubeedge
