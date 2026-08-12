#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBEEDGE_ROOT="$(cd "${PROJECT_DIR}/../../../../.." && pwd)"
PROTO="${KUBEEDGE_ROOT}/staging/src/github.com/kubeedge/api/apis/dmi/v1beta1/api.proto"
PROTO_DIR="$(dirname "${PROTO}")"
OUT_DIR="${PROJECT_DIR}/mapper_framework/generated"

PROTOC_INCLUDE="${PROTOC_INCLUDE:-/usr/local/include}"
GRPC_PYTHON_PLUGIN="${GRPC_PYTHON_PLUGIN:-grpc_python_plugin}"

command -v protoc >/dev/null 2>&1 || { echo "protoc is required" >&2; exit 1; }
command -v "${GRPC_PYTHON_PLUGIN}" >/dev/null 2>&1 || {
  echo "grpc_python_plugin is required" >&2
  exit 1
}

mkdir -p "${OUT_DIR}"
protoc \
  -I "${PROTO_DIR}" \
  -I "${PROTOC_INCLUDE}" \
  --python_out="${OUT_DIR}" \
  --grpc_python_out="${OUT_DIR}" \
  --plugin="protoc-gen-grpc_python=$(command -v "${GRPC_PYTHON_PLUGIN}")" \
  "${PROTO}"

python3 - "${OUT_DIR}/api_pb2_grpc.py" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
text = path.read_text(encoding="utf-8")
text = text.replace("import api_pb2 as api__pb2", "from . import api_pb2 as api__pb2")
path.write_text(text, encoding="utf-8")
PY

echo "Generated DMI Python protobuf files in ${OUT_DIR}"

