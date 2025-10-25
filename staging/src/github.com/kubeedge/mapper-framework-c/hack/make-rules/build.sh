#!/usr/bin/env bash
set -euo pipefail

TEMPLATE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
PARENT_DIR="$(cd "${TEMPLATE_DIR}/.." && pwd -P)"

name="${1:-${NAME:-mapper_default}}"
target="${PARENT_DIR}/${name}"

if [[ ! -d "${target}" ]]; then
  echo "Folder not found: ${target}. Run: make generate NAME=${name}"
  exit 1
fi

image="${IMAGE:-$name}"
tag="${TAG:-latest}"

echo "Building Docker image: ${image}:${tag}"
docker build -t "${image}:${tag}" -f "${target}/Dockerfile" "${target}"