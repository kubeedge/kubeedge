# As a basic image for building various components of KubeEdge
FROM ubuntu:18.04
RUN apt-get update && apt-get install -y wget \
    vim git make gcc \
    upx-ucl gcc-aarch64-linux-gnu libc6-dev-arm64-cross gcc-arm-linux-gnueabi libc6-dev-armel-cross &&\
    apt-get autoremove -y &&\
    apt-get clean &&\
    rm -rf /var/lib/apt/lists/*
RUN wget -c https://dl.google.com/go/go1.17.linux-amd64.tar.gz -O - | tar -xz -C /usr/local
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.io,direct
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin
