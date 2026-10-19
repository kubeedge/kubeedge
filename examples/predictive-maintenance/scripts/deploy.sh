#!/usr/bin/env bash
# =============================================================================
# deploy.sh — End-to-end deployment script for the Predictive Maintenance
# embodied intelligence scenario on KubeEdge.
#
# Usage:
#   ./scripts/deploy.sh [EDGE_NODE_NAME]
#
# Requirements:
#   - kubectl configured with access to your KubeEdge cluster
#   - CloudCore running on the cloud node
#   - EdgeCore running on EDGE_NODE_NAME
# =============================================================================

set -euo pipefail

EDGE_NODE="${1:-edge-node-01}"
NAMESPACE="default"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info()    { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# ---------------------------------------------------------------------------
# Step 1: Verify edge node is registered with KubeEdge
# ---------------------------------------------------------------------------
log_info "Step 1: Verifying edge node '${EDGE_NODE}' is registered..."
if ! kubectl get node "${EDGE_NODE}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q "True"; then
    log_warn "Edge node '${EDGE_NODE}' is not Ready. Make sure EdgeCore is running."
    log_warn "Continuing with deployment — manifests will apply, pods will schedule when node is ready."
fi
log_info "Edge node check complete."

# ---------------------------------------------------------------------------
# Step 2: Apply DeviceModel and Device CRDs
# ---------------------------------------------------------------------------
log_info "Step 2: Applying KubeEdge Device CRDs..."
sed "s/edge-node-01/${EDGE_NODE}/g" "${REPO_ROOT}/configs/device-model.yaml" | kubectl apply -f -
sed "s/edge-node-01/${EDGE_NODE}/g" "${REPO_ROOT}/configs/device.yaml" | kubectl apply -f -
log_info "Device CRDs applied."

# ---------------------------------------------------------------------------
# Step 3: Deploy the mapper to the edge node
# ---------------------------------------------------------------------------
log_info "Step 3: Deploying predictive maintenance mapper to edge node..."
sed "s/edge-node-01/${EDGE_NODE}/g" "${REPO_ROOT}/manifests/mapper-deployment.yaml" | kubectl apply -f -

# Wait for mapper pod to be running
log_info "Waiting for mapper pod to become Running..."
kubectl rollout status deployment/predictive-maintenance-mapper -n "${NAMESPACE}" --timeout=120s
log_info "Mapper deployed successfully."

# ---------------------------------------------------------------------------
# Step 4: Verify device twin is being updated
# ---------------------------------------------------------------------------
log_info "Step 4: Verifying device twin updates (waiting 15s for first readings)..."
sleep 15

DEVICE_STATUS=$(kubectl get device factory-sensor-01 -n "${NAMESPACE}" -o jsonpath='{.status.twins}' 2>/dev/null || echo "")
if [ -z "${DEVICE_STATUS}" ]; then
    log_warn "Device twin not yet updated. Check mapper pod logs:"
    log_warn "  kubectl logs -l app=predictive-maintenance,component=mapper -n ${NAMESPACE}"
else
    log_info "Device twin is being updated. Current status:"
    kubectl get device factory-sensor-01 -n "${NAMESPACE}" -o jsonpath='{.status.twins[*].reported.value}' | tr ' ' '\n'
fi

# ---------------------------------------------------------------------------
# Step 5: Print useful commands
# ---------------------------------------------------------------------------
echo ""
log_info "Deployment complete! Useful commands:"
echo ""
echo "  # Watch live device twin updates:"
echo "  kubectl get device factory-sensor-01 -n ${NAMESPACE} -w"
echo ""
echo "  # View mapper logs (edge AI inference output):"
echo "  kubectl logs -l app=predictive-maintenance,component=mapper -n ${NAMESPACE} -f"
echo ""
echo "  # Simulate cloud disconnect (edge autonomy test):"
echo "  ./scripts/test-edge-autonomy.sh ${EDGE_NODE}"
echo ""
echo "  # View all device twin properties:"
echo "  kubectl get device factory-sensor-01 -n ${NAMESPACE} -o yaml"
