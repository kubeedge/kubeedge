FROM ubuntu:14.04

WORKDIR /home

RUN \
apt-get update && \
apt-get install -y git wget gcc && \
mkdir golang && \
mkdir -p /home/diego-acceptance-tests && \
mkdir github.com && \
mkdir golang.org && \
wget https://dl.google.com/go/go1.11.1.linux-amd64.tar.gz && \
tar -C golang -xzf go1.11.1.linux-amd64.tar.gz && \
rm go1.11.1.linux-amd64.tar.gz

ENV PATH=$PATH:/home/golang/go/bin
ENV GOPATH=/home/
ENV GOROOT=/home/golang/go

RUN \
go get github.com/onsi/ginkgo/ginkgo && \
go get github.com/onsi/gomega/... && \
go get github.com/eclipse/paho.mqtt.golang && \
go get golang.org/x/net/websocket && \
go get golang.org/x/net/proxy && \
go get github.com/golang/glog && \
go get github.com/mattn/go-sqlite3

ENV PATH=$PATH:/home/bin

CMD ["/bin/bash"]
