# As a basic image for building various components of KubeEdge
FROM ubuntu:18.04
ARG ARCH=amd64
RUN apt-get update && apt-get install -y wget \
    vim git make gcc upx-ucl 

RUN if [ "${ARCH}" = "amd64" ]; then apt-get install -y gcc-aarch64-linux-gnu libc6-dev-arm64-cross gcc-arm-linux-gnueabihf ; fi
 
RUN apt-get autoremove -y &&\
    apt-get clean &&\
    rm -rf /var/lib/apt/lists/*

RUN if [ "${ARCH}" = "amd64" ]; then wget -c https://dl.google.com/go/go1.17.13.linux-amd64.tar.gz -O - | tar -xz -C /usr/local; \
    elif [ "${ARCH}" = "arm64" ]; then wget -c https://dl.google.com/go/go1.17.13.linux-arm64.tar.gz -O - | tar -xz -C /usr/local; \
    elif [ "${ARCH}" = "arm" ]; then wget -c https://dl.google.com/go/go1.17.13.linux-armv6l.tar.gz -O - | tar -xz -C /usr/local; \
    fi

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.io,direct
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin
