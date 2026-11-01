#!/usr/bin/env bash
# Copyright 2024 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# kind-setup.sh — bootstrap a KubeEdge edge node on a local kind cluster.
#
# Usage:
#   CLOUDCORE_IP=<cloudcore-ip> EDGE_NODE_NAME=<name> ./kind-setup.sh
#
# Required environment variables:
#   CLOUDCORE_IP     IP address of the CloudCore websocket endpoint
#   EDGE_NODE_NAME   Name for the edge node (default: edgenode1)
#   CLOUDCORE_PORT   CloudCore websocket port (default: 10000)

set -euo pipefail

EDGE_NODE_NAME="${EDGE_NODE_NAME:-edgenode1}"
CLOUDCORE_IP="${CLOUDCORE_IP:?CLOUDCORE_IP must be set}"
CLOUDCORE_PORT="${CLOUDCORE_PORT:-10000}"
NAMESPACE="kubeedge"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { echo "[kind-setup] $*"; }

# 1. Substitute placeholders in manifests and apply
log "Applying manifests for edge node: ${EDGE_NODE_NAME}"

for f in "${SCRIPT_DIR}"/*.yaml; do
  sed \
    -e "s/edgenode1/${EDGE_NODE_NAME}/g" \
    -e "s/0\.0\.0\.0:${CLOUDCORE_PORT}/${CLOUDCORE_IP}:${CLOUDCORE_PORT}/g" \
    "${f}" | kubectl apply -f -
done

# 2. Wait for the edge node pod to be Running
log "Waiting for edge node pod to be Running..."
kubectl rollout status deployment/"${EDGE_NODE_NAME}" \
  -n "${NAMESPACE}" --timeout=120s

# 3. Verify the node registers with the API server
log "Waiting for edge node '${EDGE_NODE_NAME}' to appear..."
for i in $(seq 1 30); do
  if kubectl get node "${EDGE_NODE_NAME}" &>/dev/null; then
    log "Edge node '${EDGE_NODE_NAME}' registered successfully."
    kubectl get node "${EDGE_NODE_NAME}"
    exit 0
  fi
  sleep 5
done

echo "[kind-setup] ERROR: edge node did not register within 150s." >&2
exit 1
