#!/bin/bash
set -euo pipefail

CILIUM_DS="cilium"
KUBEEDGE_NAMESPACE="kubeedge"
WORK_DIR=$(mktemp -d /tmp/kubeedge-work-XXXXXX)

trap cleanup EXIT
trap exit_trap ERR

install_yq() {
  echo "Installing yq..."
  CPU_ARCH=$(dpkg --print-architecture)
  curl -LJO https://github.com/mikefarah/yq/releases/latest/download/"yq_linux_${CPU_ARCH}" || {
    echo "[ERROR] Failed to download yq for ${CPU_ARCH}"
    exit 1
  }
  mv "yq_linux_${CPU_ARCH}" /usr/local/bin/yq
  chmod a+x /usr/local/bin/yq
  echo "yq installed successfully."
}

exit_trap() {
  local exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo "[ERROR] Command failed with exit code $exit_code: $BASH_COMMAND"
    exit $exit_code
  fi
}

print_help() {
  cat <<EOF
Usage:
  sudo $0 [OPTION] <MODE>

Options:
  -h, --help    Show this help message and exit

Modes:
  cloudcore     Apply cloud-side configurations
  edgecore      Apply edge-side configurations; must be executed on the corresponding edge node

EOF
}

check_prerequisites() {
  local mode=$1
  echo "Checking prerequisites for ${mode}..."
  if ! command -v yq &>/dev/null; then
    install_yq
  else
    yq_path=$(command -v yq)
    if [[ "$yq_path" == /snap/bin/yq ]]; then
      echo "Detected snap binary for yq at $yq_path. Reinstalling non-snap version..."
      snap remove yq
      install_yq
    else
      echo "yq is already installed at $yq_path (non-snap version)."
    fi
  fi

  case $mode in
  cloudcore)
    command -v kubectl >/dev/null || {
      echo "[ERROR] kubectl required but not found. Please install kubectl."
      exit 1
    }

    if ! kubectl get ns "$KUBEEDGE_NAMESPACE" &>/dev/null; then
      echo "[ERROR] Namespace ${KUBEEDGE_NAMESPACE} not found. Please installing KubeEdge with keadm first."
      exit 1
    fi

    if ! kubectl -n kube-system get ds "$CILIUM_DS" &>/dev/null; then
      echo "[ERROR] Cilium DaemonSet not found in kube-system namespace. Please install Cilium first."
      exit 1
    fi

    if ! kubectl -n "$KUBEEDGE_NAMESPACE" get cm cloudcore &>/dev/null; then
      echo "[ERROR] CloudCore ConfigMap not found in ${KUBEEDGE_NAMESPACE} namespace. Ensure CloudCore is properly installed with keadm."
      exit 1
    fi
    ;;

  edgecore) ;;
  esac

  echo "All prerequisites satisfied for ${mode}."
}

patch_cilium_daemonset() {
  echo "Patching main Cilium DaemonSet to exclude edge nodes..."
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
  if kubectl patch daemonset -n kube-system cilium --patch "$PATCH"; then
    echo "Cilium DaemonSet patched successfully."
  else
    echo "[ERROR] Failed to patch Cilium DaemonSet. Check kubectl permissions and cluster state."
    exit 1
  fi
}

enable_dynamic_controller() {
  echo "Enabling dynamicController in CloudCore ConfigMap..."
  FILE_NAME="${WORK_DIR}/cloudcore-configmap.yaml"

  if ! kubectl get cm -n kubeedge cloudcore -o yaml >"$FILE_NAME" 2>/dev/null; then
    echo "[ERROR] Failed to retrieve CloudCore ConfigMap. Ensure the ConfigMap exists and kubectl has appropriate permissions."
    echo "[INFO] Run 'kubectl -n kubeedge get cm cloudcore' to verify ConfigMap existence."
    exit 1
  fi

  if [ ! -s "$FILE_NAME" ]; then
    echo "[ERROR] ConfigMap file ${FILE_NAME} is empty or was not created."
    exit 1
  fi

  if ! yq e '.data."cloudcore.yaml" |= (. | fromyaml | .modules.dynamicController.enable = true | toyaml)' -i "$FILE_NAME"; then
    echo "[ERROR] Failed to update ConfigMap with yq. Check yq installation and YAML format in ${FILE_NAME}."
    exit 1
  fi

  if kubectl apply -f "${FILE_NAME}"; then
    echo "CloudCore ConfigMap updated successfully."
  else
    echo "[ERROR] Failed to apply updated ConfigMap. Check kubectl permissions and YAML content in ${FILE_NAME}."
    exit 1
  fi

  if kubectl delete pod -n kubeedge --selector=kubeedge=cloudcore; then
    echo "CloudCore pods restarted successfully."
  else
    echo "[WARNING] Failed to restart CloudCore pods. You may need to manually delete pods with 'kubectl delete pod -n kubeedge --selector=kubeedge=cloudcore'."
  fi
}

patch_cilium_rbac() {
  echo "Updating Cilium ClusterRole with KubeEdge permissions..."
  FILE_NAME="${WORK_DIR}/cilium-clusterrole.yaml"
  if ! kubectl get clusterrole cilium -o yaml >"$FILE_NAME"; then
    echo "[ERROR] Failed to retrieve Cilium ClusterRole. Ensure it exists and kubectl has permissions."
    exit 1
  fi

  if yq e '(.rules[] | select(.apiGroups[] == "cilium.io" and .resources[] == "ciliumpodippools").verbs) |= (. + ["get"] | unique)' -i "$FILE_NAME"; then
    kubectl apply -f "$FILE_NAME" || {
      echo "[ERROR] Failed to apply updated Cilium ClusterRole. Check YAML content in ${FILE_NAME}."
      exit 1
    }
  else
    echo "[ERROR] Failed to update Cilium ClusterRole with yq. Check yq installation and YAML format."
    exit 1
  fi

  FILE_NAME="${WORK_DIR}/cilium-clusterrolebinding.yaml"
  if ! kubectl get clusterrolebinding cilium -o yaml >"$FILE_NAME"; then
    echo "[ERROR] Failed to retrieve Cilium ClusterRoleBinding. Ensure it exists and kubectl has permissions."
    exit 1
  fi

  if yq e '.subjects += [{"kind": "ServiceAccount", "name": "cloudcore", "namespace": "kubeedge"}, {"kind": "ServiceAccount", "name": "cloudcore", "namespace": "default"}]' -i "$FILE_NAME"; then
    kubectl apply -f "$FILE_NAME" || {
      echo "[ERROR] Failed to apply updated Cilium ClusterRoleBinding. Check YAML content in ${FILE_NAME}."
      exit 1
    }
  else
    echo "[ERROR] Failed to update Cilium ClusterRoleBinding with yq. Check yq installation and YAML format."
    exit 1
  fi
}

update_edgecore_configuration() {
  echo "Updating edgecore configuration..."
  PATCH_FILE=/etc/kubeedge/config/edgecore.yaml

  if [ ! -f "$PATCH_FILE" ]; then
    echo "[ERROR] EdgeCore configuration file ${PATCH_FILE} not found. Ensure EdgeCore is installed."
    exit 1
  fi

  if yq e '.modules.edgeStream.enable = true' -i "$PATCH_FILE" &&
    yq e '.modules.edged.tailoredKubeletConfig.clusterDNS = ["10.96.0.10"]' -i "$PATCH_FILE" &&
    yq e '.modules.metaManager.metaServer.enable = true' -i "$PATCH_FILE"; then
    echo "EdgeCore configuration updated successfully."
  else
    echo "[ERROR] Failed to update ${PATCH_FILE} with yq. Check yq installation and YAML format."
    exit 1
  fi

  if systemctl daemon-reload && systemctl restart edgecore; then
    echo "EdgeCore service restarted successfully."
  else
    echo "[ERROR] Failed to restart EdgeCore service. Check systemctl status edgecore for details."
    exit 1
  fi
}

deploy_edge_daemonset() {
  echo "Deploying cilium-kubeedge DaemonSet for edge nodes..."
  FILE_NAME="${WORK_DIR}/cilium-kubeedge.yaml"
  if ! kubectl get daemonset -n kube-system cilium -o yaml >"$FILE_NAME"; then
    echo "[ERROR] Failed to retrieve Cilium DaemonSet. Ensure it exists and kubectl has permissions."
    exit 1
  fi

  if yq e 'del(.status)' -i "$FILE_NAME" &&
    yq e 'del(.metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"])' -i "$FILE_NAME" &&
    yq e 'del(.metadata.creationTimestamp) | del(.metadata.resourceVersion) | del(.metadata.uid)' -i "$FILE_NAME" &&
    yq e '.metadata.name = "cilium-kubeedge"' -i "$FILE_NAME"; then
    kubectl apply -f "$FILE_NAME" || {
      echo "[ERROR] Failed to apply cilium-kubeedge DaemonSet. Check YAML content in ${FILE_NAME}."
      exit 1
    }
  else
    echo "[ERROR] Failed to modify cilium-kubeedge DaemonSet with yq. Check yq installation and YAML format."
    exit 1
  fi

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
        - --k8s-api-server=127.0.0.1:10550
        - --auto-create-cilium-node-resource=true
        name: cilium-agent
      initContainers:
      - args:
        - --k8s-api-server=127.0.0.1:10550
        name: config
"
  if kubectl patch daemonset -n kube-system cilium-kubeedge --patch "$PATCH"; then
    echo "cilium-kubeedge DaemonSet patched successfully."
  else
    echo "[ERROR] Failed to patch cilium-kubeedge DaemonSet. Check kubectl permissions and cluster state."
    exit 1
  fi
}

check_sudo() {
  if [ "$EUID" -ne 0 ]; then
    echo "[ERROR] This script requires root privileges. Please run with sudo."
    exit 1
  fi
}

cleanup() {
  rm -rf "${WORK_DIR}"
}

#######################################
# KubeEdge Cilium Integration Script

# Overview:
#   This script configures Cilium CNI to work seamlessly with KubeEdge by making
#   both cloud-side and edge-side modifications.

# Key Changes:
# Cloud Side:
# - Patches main Cilium DaemonSet to exclude edge nodes
# - Enables dynamicController in CloudCore ConfigMap
# - Updates Cilium ClusterRole with KubeEdge permissions
# - Deploys cilium-kubeedge DaemonSet for edge nodes
#
# Edge Side:
# - Enables EdgeStream module for cloud-edge communication
# - Configures clusterDNS to use CoreDNS service IP
# - Enables MetaServer for edge node API access
# - Restarts EdgeCore service
#######################################
main() {
  if [[ "$(uname -s)" != "Linux" ]]; then
    echo "[ERROR] This script is only supported on Linux systems."
    exit 1
  fi

  if [[ $# -eq 1 && ("$1" == "-h" || "$1" == "--help") ]]; then
    print_help
    exit 0
  fi

  if [[ $# -ne 1 ]]; then
    echo "[ERROR] Usage: sudo $0 <cloudcore|edgecore>"
    exit 1
  fi

  check_sudo

  case "$1" in
  cloudcore)
    check_prerequisites cloudcore
    patch_cilium_daemonset
    enable_dynamic_controller
    patch_cilium_rbac
    deploy_edge_daemonset
    ;;
  edgecore)
    check_prerequisites edgecore
    update_edgecore_configuration
    ;;
  *)
    echo "[ERROR] Invalid argument: $1"
    echo "[ERROR] Usage: sudo $0 <cloudcore|edgecore>"
    exit 1
    ;;
  esac

  echo "Configuration completed successfully for $1."
}

main "$@"
