#!/usr/bin/env bash

# Batch enable/disable EdgeHub on multiple edge nodes.
# This is a simple helper that:
#   1. SSHes into each node listed in a file
#   2. Toggles EdgeHub related flags in edgecore.yaml using yq
#   3. Restarts the edgecore service
#
# Requirements on each edge node:
#   - edgecore installed and managed by systemd (service name default: edgecore)
#   - edgecore config at /etc/kubeedge/config/edgecore.yaml
#   - yq v4+ installed and available on PATH
#
# Usage:
#   batch_edgehub.sh enable nodes.txt
#   batch_edgehub.sh disable nodes.txt
#
# nodes.txt format:
#   - One SSH target per line (user@host, host, or any ssh alias)
#   - Blank lines and lines starting with # are ignored
#
# You can override defaults via environment variables:
#   EDGECORE_CONFIG   - path to edgecore.yaml (default: /etc/kubeedge/config/edgecore.yaml)
#   EDGECORE_SERVICE  - systemd service name (default: edgecore)
#
# NOTE: This script is intentionally simple and does not do fancy error
#       aggregation or retries; it stops on the first hard failure.

set -euo pipefail

EDGECORE_CONFIG="${EDGECORE_CONFIG:-/etc/kubeedge/config/edgecore.yaml}"
EDGECORE_SERVICE="${EDGECORE_SERVICE:-edgecore}"

usage() {
  cat <<EOF
Usage: $0 <enable|disable> <nodes-file>

Batch enable or disable EdgeHub on multiple edge nodes.

Arguments:
  enable|disable   Desired EdgeHub state on target nodes
  nodes-file       Path to file containing SSH targets (one per line)

Environment:
  EDGECORE_CONFIG   Path to edgecore.yaml on edge nodes (default: $EDGECORE_CONFIG)
  EDGECORE_SERVICE  edgecore systemd service name (default: $EDGECORE_SERVICE)
EOF
}

if [[ $# -ne 2 ]]; then
  usage
  exit 1
fi

ACTION="$1"
NODES_FILE="$2"

if [[ "$ACTION" != "enable" && "$ACTION" != "disable" ]]; then
  echo "error: ACTION must be 'enable' or 'disable', got '$ACTION'" >&2
  usage
  exit 1
fi

if [[ ! -f "$NODES_FILE" ]]; then
  echo "error: nodes file '$NODES_FILE' does not exist" >&2
  exit 1
fi

if ! command -v ssh >/dev/null 2>&1; then
  echo "error: ssh not found on PATH" >&2
  exit 1
fi

while IFS= read -r NODE; do
  # Skip blanks and comments
  [[ -z "$NODE" || "$NODE" =~ ^[[:space:]]*# ]] && continue

  echo "==== [$NODE] Applying EdgeHub '$ACTION' ===="

  if [[ "$ACTION" == "enable" ]]; then
    # Enable EdgeHub and websocket by default (leave QUIC as-is)
    REMOTE_CMD=$(cat <<RCMD
set -euo pipefail
if ! command -v yq >/dev/null 2>&1; then
  echo "yq not found on PATH" >&2
  exit 1
fi
if [[ ! -f "$EDGECORE_CONFIG" ]]; then
  echo "edgecore config not found at '$EDGECORE_CONFIG'" >&2
  exit 1
fi
sudo yq e -i '
  .modules.edgeHub.enable = true
  | ( .modules.edgeHub.websocket // {} ) as \$ws
  | .modules.edgeHub.websocket.enable = ( \$ws.enable // true )
' "$EDGECORE_CONFIG"
sudo systemctl restart "$EDGECORE_SERVICE"
RCMD
)
  else
    # Disable EdgeHub entirely (including both websocket and QUIC)
    REMOTE_CMD=$(cat <<RCMD
set -euo pipefail
if ! command -v yq >/dev/null 2>&1; then
  echo "yq not found on PATH" >&2
  exit 1
fi
if [[ ! -f "$EDGECORE_CONFIG" ]]; then
  echo "edgecore config not found at '$EDGECORE_CONFIG'" >&2
  exit 1
fi
sudo yq e -i '
  .modules.edgeHub.enable = false
  | ( .modules.edgeHub.websocket // {} ) as \$ws
  | .modules.edgeHub.websocket.enable = false
  | ( .modules.edgeHub.quic // {} ) as \$q
  | .modules.edgeHub.quic.enable = false
' "$EDGECORE_CONFIG"
sudo systemctl restart "$EDGECORE_SERVICE"
RCMD
)
  fi

  # Execute remote command
  ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new "$NODE" "$REMOTE_CMD"
  echo "==== [$NODE] Done ===="
  echo
done < "$NODES_FILE"

echo "All nodes processed for action '$ACTION'."

