#!/bin/bash
set -euo pipefail

CILIUM_VERSION="${CILIUM_VERSION:-v1.15.15}"
KUBEEDGE_NAMESPACE="${KUBEEDGE_NAMESPACE:-kubeedge}"
EDGE_API_SERVER="${EDGE_API_SERVER:-127.0.0.1:10550}"
CILIUM_DS="cilium"
WORK_DIR=$(mktemp -d /tmp/kubeedge-work-XXXXXX)
LOG_DIR="${WORK_DIR}/logs"
mkdir -p "$LOG_DIR"

trap 'rm -rf "$WORK_DIR"' EXIT

exit_trap() {
  local exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo "[ERROR] Command failed with exit code $exit_code: $BASH_COMMAND" >&2
    echo "Logs are available in $LOG_DIR"
    exit $exit_code
  fi
}
trap exit_trap ERR

cilium_install() {
  echo "Installing Cilium $CILIUM_VERSION..."
  cilium install --version "$CILIUM_VERSION" \
    --set encryption.enabled=true \
    --set encryption.type=wireguard \
    --set encryption.wireguard.persistentKeepalive=10s \
    --set kubeProxyReplacement=strict \
    --set enableNodePort=true
}

patch_cilium_daemonset() {
  echo "Patching Cilium DaemonSet to exclude edge nodes..."
  PATCH="
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                - key: node-role.kubernetes.io/edge
                  operator: DoesNotExist
"
  kubectl patch daemonset -n kube-system "$CILIUM_DS" --patch "$PATCH"
}

patch_cilium_rbac() {
  echo "Updating Cilium RBAC for KubeEdge compatibility..."
  FILE_NAME="${WORK_DIR}/cilium-clusterrole.yaml"
  kubectl get clusterrole cilium -o yaml >"$FILE_NAME"
  yq e '(.rules[] | select(.apiGroups[] == "cilium.io" and .resources[] == "ciliumpodippools").verbs) |= (. + ["get"] | unique)' -i "$FILE_NAME"
  kubectl apply -f "$FILE_NAME"

  FILE_NAME="${WORK_DIR}/cilium-clusterrolebinding.yaml"
  kubectl get clusterrolebinding cilium -o yaml >"$FILE_NAME"
  yq e '.subjects += [{"kind": "ServiceAccount", "name": "cloudcore", "namespace": "'"$KUBEEDGE_NAMESPACE"'"}, {"kind": "ServiceAccount", "name": "cloudcore", "namespace": "default"}]' -i "$FILE_NAME"
  kubectl apply -f "$FILE_NAME"
}

deploy_edge_daemonset() {
  echo "Deploying cilium-kubeedge DaemonSet for edge nodes..."
  FILE_NAME="${WORK_DIR}/cilium-kubeedge.yaml"
  kubectl get daemonset -n kube-system "$CILIUM_DS" -o yaml >"$FILE_NAME"
  yq e 'del(.status)' -i "$FILE_NAME"
  yq e 'del(.metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"])' -i "$FILE_NAME"
  yq e 'del(.metadata.creationTimestamp) | del(.metadata.resourceVersion) | del(.metadata.uid)' -i "$FILE_NAME"
  yq e '.metadata.name = "cilium-kubeedge"' -i "$FILE_NAME"
  kubectl apply -f "$FILE_NAME"
  PATCH="
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                - key: node-role.kubernetes.io/edge
                  operator: Exists
      containers:
      - args:
        - --config-dir=/tmp/cilium/config-map
        - --k8s-api-server=$EDGE_API_SERVER
        - --auto-create-cilium-node-resource=true
        name: cilium-agent
      initContainers:
      - args:
        - --k8s-api-server=$EDGE_API_SERVER
        name: config
"
  kubectl patch daemonset -n kube-system cilium-kubeedge --patch "$PATCH"
}

deploy_busybox() {
  echo "Deploying BusyBox DaemonSet..."
  kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: busybox
spec:
  selector:
    matchLabels:
      app: busybox
  template:
    metadata:
      labels:
        app: busybox
    spec:
      containers:
      - image: busybox
        command: ["sleep", "3600"]
        imagePullPolicy: IfNotPresent
        name: busybox
EOF

  echo "Waiting for BusyBox pods to be ready..."
  kubectl wait --for=condition=Ready pod -l app=busybox --timeout=300s
}

ping_test() {
  echo "Running ping tests between BusyBox pods..."
  kubectl get pods -l app=busybox -o jsonpath='{.items[*].status.podIP}' >"${WORK_DIR}/busybox-ips.txt"
  echo "BusyBox pod IPs:"
  cat "${WORK_DIR}/busybox-ips.txt"

  IPS=$(cat "${WORK_DIR}/busybox-ips.txt")
  if [ -z "$IPS" ]; then
    echo "No BusyBox pod IPs found. Exiting with failure."
    exit 1
  fi

  FAILED=false
  for POD in $(kubectl get pods -l app=busybox -o name); do
    POD_IP=$(kubectl get "$POD" -o jsonpath='{.status.podIP}')
    for IP in $IPS; do
      if [ "$POD_IP" != "$IP" ]; then
        echo "Pinging $IP from $POD..."
        LOG_FILE="${LOG_DIR}/ping-${POD##*/}-${IP//./-}.log"
        kubectl exec "$POD" -- sh -c "ping -c 4 $IP" >"$LOG_FILE" 2>&1
        EXIT_CODE=$?
        if [ $EXIT_CODE -ne 0 ]; then
          echo "Ping from $POD to $IP failed with exit code $EXIT_CODE."
          cat "$LOG_FILE"
          FAILED=true
        else
          echo "Ping from $POD to $IP succeeded."
          cat "$LOG_FILE"
        fi
      fi
    done
  done

  if [ "$FAILED" = true ]; then
    echo "One or more ping tests failed. Exiting with failure."
    exit 1
  fi
  echo "All ping tests succeeded."
}

update_cilium_operator() {
  kubectl patch deployment cilium-operator \
    -n kube-system \
    --type='merge' \
    -p '{
    "spec": {
      "template": {
        "spec": {
          "nodeSelector": {
            "kubernetes.io/hostname": "test-control-plane"
          }
        }
      }
    }
  }'

  kubectl delete pod -n kube-system -l app.kubernetes.io/name=cilium-operator --force
}

main() {
  CONTAINER_RUNTIME="containerd" ENABLE_DAEMON=true ENABLE_CNI=true bash -x ./hack/local-up-kubeedge.sh || {
    echo "Failed to start cluster!"
    exit 1
  }
  cilium_install
  update_cilium_operator
  patch_cilium_daemonset
  patch_cilium_rbac
  deploy_edge_daemonset
  cilium status --wait
  deploy_busybox
  ping_test
}

main
