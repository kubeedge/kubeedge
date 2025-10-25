FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get purge -y libprotobuf-dev protobuf-compiler || true

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake pkg-config git ca-certificates \
    libyaml-dev libmicrohttpd-dev libcjson-dev libcurl4-openssl-dev \
    libmysqlclient-dev default-libmysqlclient-dev \
    libhiredis-dev \
    libmosquitto-dev \
    libavformat-dev libavcodec-dev libavutil-dev libswscale-dev \
    libswresample-dev libpostproc-dev \
    libssl-dev \
    wget autoconf automake libtool curl unzip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp

RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v31.1/protobuf-31.1.tar.gz \
    && tar -xzf protobuf-31.1.tar.gz \
    && cd protobuf-31.1 \
    && cmake -B build -DCMAKE_BUILD_TYPE=Release -Dprotobuf_BUILD_TESTS=OFF . \
    && cmake --build build -j$(nproc) \
    && cmake --install build \
    && ldconfig \
    && cd .. && rm -rf protobuf-31.1 protobuf-31.1.tar.gz

RUN git clone --branch v1.49.0 --depth=1 https://github.com/grpc/grpc \
    && cd grpc \
    && git submodule update --init --recursive \
    && mkdir -p cmake/build \
    && cd cmake/build \
    && cmake -DgRPC_BUILD_TESTS=OFF -DCMAKE_BUILD_TYPE=Release ../.. \
    && make -j$(nproc) \
    && make install \
    && ldconfig \
    && cd ../../.. && rm -rf grpc

RUN git clone --depth=1 https://github.com/protobuf-c/protobuf-c.git \
    && cd protobuf-c \
    && ./autogen.sh \
    && ./configure \
    && make -j$(nproc) \
    && make install \
    && ldconfig \
    && cd .. && rm -rf protobuf-c

WORKDIR /app
COPY . .
RUN protoc --proto_path=/app/dmi/v1beta1 \
           --cpp_out=/app/dmi/v1beta1 \
           --grpc_out=/app/dmi/vn1beta1 \
           --plugin=protoc-gen-grpc=`which grpc_cpp_plugin` \
           /app/dmi/v1beta1/api.proto

WORKDIR /app
RUN mkdir -p build && cd build && cmake .. && make -j$(nproc)

ENV EDGECORE_SOCK=/etc/kubeedge/dmi.sock
CMD ["/app/build/main", "/app/config.yaml"]
