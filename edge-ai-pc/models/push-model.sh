#!/usr/bin/env bash
## push-model.sh — Push ONNX/GGUF/OpenVINO models as OCI artifacts to registry
## Tool: ORAS (OCI Registry As Storage) v1.2.0+
## Install: https://oras.land/docs/installation
##
## Usage:
##   bash push-model.sh yolov8n   /path/to/yolov8n_openvino_model  openvino-int8
##   bash push-model.sh phi3-mini /path/to/Phi-3-mini-4k-instruct-q4.gguf q4-km
##   bash push-model.sh list      # list all pushed models

set -euo pipefail

REGISTRY="${MODEL_REGISTRY:-registry.example.com}"    # override via env var
REGISTRY_USER="${REGISTRY_USER:-}"
REGISTRY_PASS="${REGISTRY_PASS:-}"

MODEL_NAME="${1:-}"
MODEL_PATH="${2:-}"
MODEL_TAG="${3:-latest}"

ORAS_VERSION="v1.2.0"

# ─── Helpers ──────────────────────────────────────────────────────────────────
log()  { echo "[ INFO ] $*"; }
err()  { echo "[ ERR  ] $*" >&2; exit 1; }
warn() { echo "[ WARN ] $*"; }

check_oras() {
  if ! command -v oras &>/dev/null; then
    log "ORAS not found. Installing..."
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m); [[ "$ARCH" == "x86_64" ]] && ARCH="amd64"
    curl -LO "https://github.com/oras-project/oras/releases/download/${ORAS_VERSION}/oras_${ORAS_VERSION#v}_${OS}_${ARCH}.tar.gz"
    tar xzf oras_*.tar.gz oras
    sudo mv oras /usr/local/bin/
    rm -f oras_*.tar.gz
    log "ORAS installed: $(oras version)"
  fi
}

registry_login() {
  if [[ -n "$REGISTRY_USER" && -n "$REGISTRY_PASS" ]]; then
    echo "$REGISTRY_PASS" | oras login "$REGISTRY" -u "$REGISTRY_USER" --password-stdin
    log "Logged in to $REGISTRY"
  else
    warn "REGISTRY_USER / REGISTRY_PASS not set. Assuming anonymous or already logged in."
  fi
}

# ─── Push OpenVINO model directory ────────────────────────────────────────────
push_openvino() {
  local name="$1" path="$2" tag="$3"
  local ref="${REGISTRY}/models/${name}:${tag}"

  [[ -d "$path" ]] || err "Directory not found: $path"

  log "Pushing OpenVINO model directory: $path → $ref"
  # Find .xml and .bin files (OpenVINO IR format)
  local xml; xml=$(find "$path" -name "*.xml" | head -1)
  local bin; bin=$(find "$path" -name "*.bin" | head -1)

  [[ -f "$xml" ]] || err "No .xml found in $path"
  [[ -f "$bin" ]] || err "No .bin found in $path"

  oras push "$ref" \
    "${xml}:application/vnd.kubeedge.openvino.xml.v1" \
    "${bin}:application/vnd.kubeedge.openvino.bin.v1" \
    --annotation "org.opencontainers.image.title=${name}" \
    --annotation "org.opencontainers.image.description=OpenVINO IR model" \
    --annotation "kubeedge.io/model.format=openvino" \
    --annotation "kubeedge.io/model.precision=int8"

  log "✅ Pushed: $ref"
}

# ─── Push GGUF model file ──────────────────────────────────────────────────────
push_gguf() {
  local name="$1" path="$2" tag="$3"
  local ref="${REGISTRY}/models/${name}:${tag}"

  [[ -f "$path" ]] || err "File not found: $path"

  local size_gb
  size_gb=$(du -sh "$path" | cut -f1)
  log "Pushing GGUF model: $path ($size_gb) → $ref"

  oras push "$ref" \
    "${path}:application/vnd.kubeedge.gguf.v1" \
    --annotation "org.opencontainers.image.title=${name}" \
    --annotation "org.opencontainers.image.description=GGUF quantized LLM" \
    --annotation "kubeedge.io/model.format=gguf" \
    --annotation "kubeedge.io/model.quantization=${tag}"

  log "✅ Pushed: $ref"
}

# ─── Pull a model (for verification) ─────────────────────────────────────────
pull_model() {
  local name="$1" tag="${2:-latest}"
  local ref="${REGISTRY}/models/${name}:${tag}"
  local out_dir="/tmp/model-verify/${name}-${tag}"
  mkdir -p "$out_dir"

  log "Pulling $ref → $out_dir"
  oras pull "$ref" -o "$out_dir"
  ls -lh "$out_dir"
  log "✅ Pull verified: $ref"
}

# ─── List all models in registry ─────────────────────────────────────────────
list_models() {
  log "Listing models in ${REGISTRY}/models:"
  oras repo tags "${REGISTRY}/models" 2>/dev/null || \
    warn "Registry does not support tag listing or is unauthenticated."
}

# ─── Main ─────────────────────────────────────────────────────────────────────
check_oras
registry_login

case "$MODEL_NAME" in
  yolov8n|yolov8*)
    push_openvino "$MODEL_NAME" "$MODEL_PATH" "$MODEL_TAG"
    ;;
  phi3*|llama*|mistral*|gemma*)
    push_gguf "$MODEL_NAME" "$MODEL_PATH" "$MODEL_TAG"
    ;;
  pull)
    pull_model "$MODEL_PATH" "$MODEL_TAG"
    ;;
  list)
    list_models
    ;;
  *)
    echo "Usage:"
    echo "  bash push-model.sh <model-name> <path> <tag>"
    echo "  bash push-model.sh list"
    echo ""
    echo "Examples:"
    echo "  bash push-model.sh yolov8n   ./yolov8n_openvino_model  openvino-int8"
    echo "  bash push-model.sh phi3-mini ./Phi-3-mini-4k.gguf      q4-km"
    echo "  bash push-model.sh list"
    exit 1
    ;;
esac
