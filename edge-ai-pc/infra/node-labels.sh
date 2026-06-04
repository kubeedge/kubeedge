#!/usr/bin/env bash
## node-labels.sh — Label and optionally taint AI PC edge nodes
## Usage:
##   bash node-labels.sh intel ai-pc-node-01     # Intel NPU node
##   bash node-labels.sh nvidia ai-pc-node-02    # NVIDIA GPU node
##   bash node-labels.sh amd ai-pc-node-03       # AMD Ryzen AI node
##   bash node-labels.sh remove ai-pc-node-01    # Remove all AI-PC labels

set -euo pipefail

VENDOR="${1:-intel}"
NODE="${2:-ai-pc-node-01}"

# ─── Common labels for every AI PC edge node ──────────────────────────────
apply_common_labels() {
  kubectl label node "$NODE" \
    node-role.kubernetes.io/edge="" \
    kubeedge.io/edge-node="true" \
    node-type="ai-pc" \
    edge-capability="inference" \
    --overwrite

  echo "✅ Common labels applied to $NODE"
}

# ─── Intel NPU (Core Ultra / Meteor Lake / Lunar Lake) ────────────────────
apply_intel_npu_labels() {
  kubectl label node "$NODE" \
    feature.node.kubernetes.io/npu="intel" \
    accelerator="intel-npu" \
    vendor="intel" \
    openvino.intel.com/npu="true" \
    --overwrite

  echo "✅ Intel NPU labels applied to $NODE"
  echo "   Resource exposed: gpu.intel.com/i915"
}

# ─── NVIDIA GPU ──────────────────────────────────────────────────────────
apply_nvidia_gpu_labels() {
  kubectl label node "$NODE" \
    accelerator="nvidia-gpu" \
    vendor="nvidia" \
    nvidia.com/gpu.present="true" \
    --overwrite

  echo "✅ NVIDIA GPU labels applied to $NODE"
  echo "   Resource exposed: nvidia.com/gpu"
}

# ─── AMD Ryzen AI NPU ────────────────────────────────────────────────────
apply_amd_labels() {
  kubectl label node "$NODE" \
    feature.node.kubernetes.io/npu="amd" \
    accelerator="amd-ryzen-ai" \
    vendor="amd" \
    --overwrite

  echo "✅ AMD Ryzen AI labels applied to $NODE"
  echo "   Resource exposed: amd.com/npu (via AMD device plugin)"
}

# ─── Remove all AI-PC labels ─────────────────────────────────────────────
remove_labels() {
  kubectl label node "$NODE" \
    node-role.kubernetes.io/edge- \
    kubeedge.io/edge-node- \
    node-type- \
    edge-capability- \
    feature.node.kubernetes.io/npu- \
    accelerator- \
    vendor- \
    openvino.intel.com/npu- \
    nvidia.com/gpu.present- \
    --overwrite 2>/dev/null || true

  echo "✅ All AI-PC labels removed from $NODE"
}

# ─── Optional: taint node so only AI workloads schedule here ──────────────
apply_taint() {
  read -r -p "Apply taint (edge-ai=true:NoSchedule) to restrict non-AI workloads? [y/N] " yn
  case $yn in
    [Yy]*)
      kubectl taint node "$NODE" edge-ai=true:NoSchedule --overwrite
      echo "✅ Taint applied. Add toleration to AI workload pods:"
      echo "   tolerations:"
      echo "   - key: edge-ai"
      echo "     operator: Equal"
      echo "     value: 'true'"
      echo "     effect: NoSchedule"
      ;;
    *) echo "ℹ️  Taint skipped." ;;
  esac
}

# ─── Main ─────────────────────────────────────────────────────────────────
echo "═══════════════════════════════════════════"
echo "  AI PC Node Labeler — Target: $NODE"
echo "  Vendor: $VENDOR"
echo "═══════════════════════════════════════════"

# Verify node exists
if ! kubectl get node "$NODE" &>/dev/null; then
  echo "❌ Node '$NODE' not found in cluster. Is EdgeCore running?"
  exit 1
fi

case "$VENDOR" in
  intel)  apply_common_labels; apply_intel_npu_labels ;;
  nvidia) apply_common_labels; apply_nvidia_gpu_labels ;;
  amd)    apply_common_labels; apply_amd_labels ;;
  remove) remove_labels ;;
  *)
    echo "❌ Unknown vendor: $VENDOR. Use: intel | nvidia | amd | remove"
    exit 1
    ;;
esac

apply_taint

echo ""
echo "Current node labels:"
kubectl get node "$NODE" --show-labels
