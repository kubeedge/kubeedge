#!/usr/bin/env bash
set -euo pipefail

TEMPLATE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
PARENT_DIR="$(cd "${TEMPLATE_DIR}/.." && pwd -P)"

if [[ $# -lt 2 ]] || [[ -z "${1:-}" ]] || [[ -z "${2:-}" ]]; then
  read -p "Please input the mapper name (like 'Bluetooth', 'BLE'): " name
  if [[ -z "${name}" ]]; then
    echo "the mapper name is required"
    exit 1
  fi
  read -p "Please input the build method (stream/nostream)(stream currently not supported): " buildMethod
  if [[ -z "${buildMethod}" ]]; then
    echo "the build method is required"
    exit 1
  fi
else
  name="$1"
  buildMethod="$2"
fi

target="${PARENT_DIR}/${name}"

echo "[generate] TEMPLATE_DIR=${TEMPLATE_DIR}"
echo "[generate] TARGET_DIR=${target}"

if [[ -e "${target}" ]]; then
  echo "Target folder already exists: ${target}"
  exit 1
fi
mkdir -p "${target}"

if command -v rsync >/dev/null 2>&1; then
  rsync -a --delete \
    --exclude '/build' \
    --exclude '/.git' \
    --exclude '/mappers' \
    "${TEMPLATE_DIR}/" "${target}/"
else
  ( cd "${TEMPLATE_DIR}" && tar --exclude='./build' --exclude='./.git' --exclude='./mappers' -cf - . ) | ( cd "${target}" && tar -xf - )
fi

cat > "${target}/Dockerfile" <<'EOF'
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
EOF

cat > "${target}/.dockerignore" <<'EOF'
build
.git
mappers
*.log
*.swp
*.swo
EOF

echo "Mapper project generated at: ${target}"
echo "Build method: ${buildMethod}"
echo "You can now run: make build NAME=${name}"
