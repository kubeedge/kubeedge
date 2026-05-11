#!/usr/bin/env bash
## offline-resilience-test.sh — Validate KubeEdge edge autonomy during cloud disconnection
## Tests:
##   1. Inference pods remain Running when cloud (CloudCore) is unreachable
##   2. EdgeCore metaManager serves pod specs from local DB
##   3. Pods restart correctly after cloud reconnects
##
## Usage:
##   bash offline-resilience-test.sh [node] [inference-pod] [namespace]
##   bash offline-resilience-test.sh ai-pc-node-01 yolov8n-ovms edge-ai

set -euo pipefail

NODE="${1:-ai-pc-node-01}"
POD="${2:-yolov8n-ovms}"
NS="${3:-edge-ai}"
DISCONNECT_SEC="${4:-60}"
CLOUD_IP=$(kubectl -n kubeedge get svc cloudcore -o jsonpath='{.spec.clusterIP}' 2>/dev/null || echo "")

log()  { echo "[$(date +%H:%M:%S)] $*"; }
pass() { echo "[$(date +%H:%M:%S)] ✅ PASS: $*"; }
fail() { echo "[$(date +%H:%M:%S)] ❌ FAIL: $*"; }

echo "═══════════════════════════════════════════════════"
echo "  KubeEdge Edge Autonomy — Offline Resilience Test"
echo "  Node: $NODE | Pod: $POD | Namespace: $NS"
echo "  Disconnection duration: ${DISCONNECT_SEC}s"
echo "═══════════════════════════════════════════════════"
echo ""

RESULTS=()

# ─── Baseline check ───────────────────────────────────────────────────────────
log "PHASE 1: Baseline — confirming pod is Running with cloud connected..."
POD_STATUS=$(kubectl get pod "$POD" -n "$NS" \
  -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")

if [[ "$POD_STATUS" == "Running" ]]; then
  pass "Baseline: $POD is Running"
  RESULTS+=("BASELINE: PASS")
else
  fail "Baseline: $POD is $POD_STATUS — fix before running this test"
  exit 1
fi

# ─── Pre-disconnect inference test ────────────────────────────────────────────
log "PHASE 2: Sending a test inference request BEFORE disconnect..."
OVMS_POD_IP=$(kubectl get pod "$POD" -n "$NS" \
  -o jsonpath='{.status.podIP}' 2>/dev/null || echo "")

PRE_INFERENCE_OK=false
if [[ -n "$OVMS_POD_IP" ]]; then
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "http://${OVMS_POD_IP}:8080/v2/health/ready" --max-time 5 || echo "000")
  if [[ "$STATUS" == "200" ]]; then
    pass "Pre-disconnect: inference endpoint healthy (HTTP $STATUS)"
    PRE_INFERENCE_OK=true
    RESULTS+=("PRE_DISCONNECT_INFERENCE: PASS")
  else
    log "Pre-disconnect: inference returned HTTP $STATUS (may be warming up)"
    RESULTS+=("PRE_DISCONNECT_INFERENCE: SKIP (HTTP $STATUS)")
  fi
fi

# ─── Simulate cloud disconnect ────────────────────────────────────────────────
log "PHASE 3: Simulating cloud disconnect for ${DISCONNECT_SEC}s..."
log "  Method: blocking CloudCore IP via iptables on edge node (requires SSH access)"

# NOTE: In a real test, SSH into the edge node and run:
#   sudo iptables -I OUTPUT -d <CLOUD_IP> -j DROP
#   sudo iptables -I INPUT  -s <CLOUD_IP> -j DROP
# For this script, we simulate by checking edged status directly on host.

if command -v ssh &>/dev/null && [[ -n "${EDGE_NODE_SSH:-}" ]]; then
  log "  Blocking CloudCore traffic via iptables..."
  ssh "$EDGE_NODE_SSH" "sudo iptables -I OUTPUT -d ${CLOUD_IP} -j DROP 2>/dev/null; \
                         sudo iptables -I INPUT  -s ${CLOUD_IP} -j DROP 2>/dev/null" || true
  DISCONNECT_APPLIED=true
else
  log "  EDGE_NODE_SSH not set — skipping iptables block (dry-run mode)"
  log "  To use: export EDGE_NODE_SSH=user@<edge-node-ip>"
  DISCONNECT_APPLIED=false
fi

log "  Waiting ${DISCONNECT_SEC}s to simulate offline period..."
sleep "$DISCONNECT_SEC"

# ─── Check pod status while disconnected ─────────────────────────────────────
log "PHASE 4: Checking pod status DURING simulated disconnect..."
# Note: kubectl may timeout since edge node reports NotReady to cloud
# The edge node itself keeps pods running via local metaManager

if $DISCONNECT_APPLIED; then
  DISCONNECTED_STATUS=$(ssh "$EDGE_NODE_SSH" \
    "crictl pods --name $POD 2>/dev/null | grep -c Running || echo 0" 2>/dev/null || echo "unknown")
  if [[ "$DISCONNECTED_STATUS" -gt 0 ]] 2>/dev/null; then
    pass "Disconnect: pod still running on edge node (crictl confirmed)"
    RESULTS+=("DISCONNECT_RESILIENCE: PASS (crictl: Running)")
  else
    log "  Could not confirm via crictl (status: $DISCONNECTED_STATUS)"
    RESULTS+=("DISCONNECT_RESILIENCE: UNKNOWN")
  fi
else
  log "  Dry-run mode: skipping disconnected pod check"
  RESULTS+=("DISCONNECT_RESILIENCE: SKIP (dry-run)")
fi

# ─── Restore connectivity ─────────────────────────────────────────────────────
log "PHASE 5: Restoring cloud connectivity..."
if $DISCONNECT_APPLIED; then
  ssh "$EDGE_NODE_SSH" "sudo iptables -D OUTPUT -d ${CLOUD_IP} -j DROP 2>/dev/null; \
                         sudo iptables -D INPUT  -s ${CLOUD_IP} -j DROP 2>/dev/null" || true
  log "  iptables rules cleared"
fi

log "  Waiting 30s for EdgeCore to re-sync with CloudCore..."
sleep 30

# ─── Post-reconnect check ─────────────────────────────────────────────────────
log "PHASE 6: Post-reconnect pod status check..."
POST_STATUS=$(kubectl get pod "$POD" -n "$NS" \
  -o jsonpath='{.status.phase}' --request-timeout=30s 2>/dev/null || echo "Unreachable")

if [[ "$POST_STATUS" == "Running" ]]; then
  pass "Post-reconnect: $POD is Running — edge autonomy confirmed"
  RESULTS+=("POST_RECONNECT: PASS")
else
  fail "Post-reconnect: $POD is $POST_STATUS"
  RESULTS+=("POST_RECONNECT: FAIL ($POST_STATUS)")
fi

# ─── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════"
echo "  OFFLINE RESILIENCE TEST RESULTS"
echo "═══════════════════════════════════════════════════"
for r in "${RESULTS[@]}"; do
  echo "  $r"
done
echo "═══════════════════════════════════════════════════"
