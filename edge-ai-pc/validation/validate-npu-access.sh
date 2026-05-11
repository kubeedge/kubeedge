#!/usr/bin/env bash
## validate-npu-access.sh — Verify NPU device is visible inside a Kubernetes pod
## Usage:
##   bash validate-npu-access.sh [node-name] [namespace]

set -euo pipefail

NODE="${1:-ai-pc-node-01}"
NS="${2:-default}"
TEST_POD="npu-validate-$(date +%s)"

echo "[ 1/4 ] Checking node allocatable resources..."
kubectl get node "$NODE" -o json | python3 -c "
import sys, json
node = json.load(sys.stdin)
alloc = node['status'].get('allocatable', {})
npu = alloc.get('gpu.intel.com/i915', '0')
print(f'  gpu.intel.com/i915 allocatable: {npu}')
if int(npu) == 0:
    print('  ERROR: NPU resource not allocatable. Is intel-npu-plugin running?'); sys.exit(1)
else:
    print('  OK: NPU resource available')
"

echo ""
echo "[ 2/4 ] Verifying intel-npu-plugin pod on node..."
PLUGIN_POD=$(kubectl -n kube-system get pods -l app=intel-npu-plugin \
  --field-selector "spec.nodeName=${NODE}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [[ -z "$PLUGIN_POD" ]]; then
  echo "  ERROR: No intel-npu-plugin pod on $NODE"; exit 1
fi
echo "  OK: Plugin pod $PLUGIN_POD running"

echo ""
echo "[ 3/4 ] Launching test pod requesting gpu.intel.com/i915..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $TEST_POD
  namespace: $NS
spec:
  restartPolicy: Never
  nodeSelector:
    kubernetes.io/hostname: $NODE
  tolerations:
  - key: node-role.kubernetes.io/edge
    operator: Exists
    effect: NoSchedule
  containers:
  - name: validator
    image: openvino/ubuntu22_dev:2024.3.0
    command: [sh, -c]
    args:
    - |
      ls -la /dev/accel/ 2>/dev/null || echo "WARN: /dev/accel not found"
      python3 -c "
      from openvino.runtime import Core
      devices = Core().available_devices
      print('Devices:', devices)
      print('NPU OK' if 'NPU' in devices else 'NPU MISSING')
      "
    resources:
      limits:
        gpu.intel.com/i915: "1"
        memory: 512Mi
      requests:
        cpu: 100m
        memory: 256Mi
EOF

echo ""
echo "[ 4/4 ] Waiting for pod completion (max 120s)..."
kubectl wait --for=condition=Ready pod/"$TEST_POD" -n "$NS" --timeout=90s 2>/dev/null || true
kubectl -n "$NS" logs "$TEST_POD" 2>/dev/null || echo "(logs pending)"
echo ""
read -r -p "Delete test pod? [Y/n] " yn
[[ "$yn" =~ ^[Nn] ]] || kubectl -n "$NS" delete pod "$TEST_POD" --grace-period=0
echo "Done."
