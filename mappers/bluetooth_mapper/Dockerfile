FROM ubuntu:16.04

RUN mkdir -p kubeedge

COPY . kubeedge/

WORKDIR kubeedge

ENTRYPOINT ["/kubeedge/main","--logtostderr=true"]
