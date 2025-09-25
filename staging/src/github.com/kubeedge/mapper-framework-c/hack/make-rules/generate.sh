#!/usr/bin/env bash
set -euo pipefail

TEMPLATE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
PARENT_DIR="$(cd "${TEMPLATE_DIR}/.." && pwd -P)"

name="${1:-${NAME:-mapper_default}}"
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
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake pkg-config git ca-certificates \
    libyaml-dev libmicrohttpd-dev libcjson-dev libcurl4-openssl-dev \
    libhiredis-dev libmysqlclient-dev \
    libprotobuf-c-dev protobuf-c-compiler \
    libprotobuf-dev protobuf-compiler protobuf-compiler-grpc \
    libgrpc-dev libgrpc++-dev \
    libmosquitto-dev \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .
RUN mkdir -p build && cd build && cmake .. && make -j

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