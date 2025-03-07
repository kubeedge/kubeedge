#!/bin/bash
set -euo pipefail

CILIUM_DS="cilium"
KUBEEDGE_NAMESPACE="kubeedge"
WORK_DIR=$(mktemp -d /tmp/kubeedge-work-XXXXXX)

trap cleanup EXIT
trap exit_trap ERR

install_yq() {
  CPU_ARCH=$(dpkg --print-architecture)
  # install yq
  curl -LJO https://github.com/mikefarah/yq/releases/latest/download/"yq_linux_${CPU_ARCH}"
  mv "yq_linux_${CPU_ARCH}" /usr/local/bin/yq
  chmod a+x /usr/local/bin/yq
}

exit_trap() {
  local exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo "[ERROR] Command failed with exit code $exit_code: $BASH_COMMAND" >&2
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
  if ! command -v yq &>/dev/null; then
    install_yq
  fi

  case $mode in
  cloudcore)
    command -v kubectl >/dev/null || {
      echo >&2 "kubectl required but not found. Aborting."
      exit 1
    }

    if ! kubectl get ns "$KUBEEDGE_NAMESPACE" &>/dev/null; then
      echo "No workload running in KubeEdge namespace, please install KubeEdge CloudCore and EdgeCore 1st."
      exit 1
    fi

    if ! kubectl -n kube-system get ds "$CILIUM_DS" &>/dev/null; then
      echo "Cilium DaemonSet not found. Please install Cilium first."
      exit 1
    fi
    ;;

  edgecore) ;;
  esac

  echo "All prerequisites satisfied for ${mode}."
}

patch_cilium_daemonset() {
  echo "Patches main Cilium DaemonSet to exclude edge nodes..."
  # apply patch to daemonset
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
  kubectl patch daemonset -n kube-system cilium --patch "$PATCH"
}

enable_dynamic_controller() {
  echo "Enables dynamicController in CloudCore ConfigMap..."
  FILE_NAME="${WORK_DIR}/cloudcore-configmap.yaml"
  kubectl get cm -n kubeedge cloudcore -o yaml >"$FILE_NAME"
  yq e '.data."cloudcore.yaml" |= (. | fromyaml | .modules.dynamicController.enable = true | toyaml)' -i "$FILE_NAME"

  kubectl apply -f "${FILE_NAME}"
  kubectl delete pod -n kubeedge --selector=kubeedge=cloudcore
}

patch_cilium_rbac() {
  echo "Updates Cilium ClusterRole with KubeEdge permissions..."
  FILE_NAME="${WORK_DIR}"/cilium-clusterrole.yaml
  kubectl get clusterrole cilium -o yaml >"$FILE_NAME"
  yq e '(.rules[] | select(.apiGroups[] == "cilium.io" and .resources[] == "ciliumpodippools").verbs) |= (. + ["get"] | unique)' -i "$FILE_NAME"
  kubectl apply -f "$FILE_NAME"

  FILE_NAME="${WORK_DIR}"/cilium-clusterrolebinding.yaml
  kubectl get clusterrolebinding cilium -o yaml >"$FILE_NAME"
  yq e '.subjects += [{"kind": "ServiceAccount", "name": "cloudcore", "namespace": "kubeedge"}, {"kind": "ServiceAccount", "name": "cloudcore", "namespace": "default"}]' -i "$FILE_NAME"
  kubectl apply -f "$FILE_NAME"
}

update_edgecore_configuration() {
  echo "Updating edgecore configuration..."

  # FEAT: support configure edgecore in cloud side
  # https://github.com/kubeedge/kubeedge/pull/6049

  PATCH_FILE=/etc/kubeedge/config/edgecore.yaml

  yq e '.modules.edgeStream.enable = true' -i "$PATCH_FILE"
  yq e '.modules.edged.tailoredKubeletConfig.clusterDNS = ["10.96.0.10"]' -i "$PATCH_FILE"
  yq e '.modules.metaManager.metaServer.enable = true' -i "$PATCH_FILE"
  yq e '.modules.serviceBus.enable = true' -i "$PATCH_FILE"

  systemctl daemon-reload
  systemctl restart edgecore
}

deploy_edge_daemonset() {
  echo "Deploys cilium-kubeedge DaemonSet for edge nodes..."
  FILE_NAME="${WORK_DIR}"/cilium-kubeedge.yaml
  kubectl get daemonset -n kube-system cilium -o yaml >"$FILE_NAME"
  # delete status
  yq e 'del(.status)' -i "$FILE_NAME"
  yq e 'del(.metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"])' -i "$FILE_NAME"
  yq e 'del(.metadata.creationTimestamp) | del(.metadata.resourceVersion) | del(.metadata.uid)' -i "$FILE_NAME"
  # rename cilium -> cilium-kubeedge
  yq e '.metadata.name = "cilium-kubeedge"' -i "$FILE_NAME"
  # create daemonset cilium-kubeedge
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
        - --k8s-api-server=127.0.0.1:10550
        - --auto-create-cilium-node-resource=true
        name: cilium-agent
      initContainers:
      - args:
        - --k8s-api-server=127.0.0.1:10550
        name: config
"
  kubectl patch daemonset -n kube-system cilium-kubeedge --patch "$PATCH"
}

check_sudo() {
  if [ "$EUID" -ne 0 ]; then
    echo "This script requires root privileges. Please run with sudo."
    exit 1
  fi
}

cleanup() {
  rm -rf ${WORK_DIR}
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
# - Enables ServiceBus for edge services
# - Restarts EdgeCore service
#######################################
main() {
  if [[ "$(uname -s)" != "Linux" ]]; then
    echo "This script is only supported on Linux systems."
    exit 1
  fi

  if [[ $# -eq 1 && ("$1" == "-h" || "$1" == "--help") ]]; then
    print_help
    exit 0
  fi

  if [[ $# -ne 1 ]]; then
    echo "Usage: sudo $0 <cloudcore|edgecore>"
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
    cleanup
    ;;
  edgecore)
    check_prerequisites edgecore
    update_edgecore_configuration
    cleanup
    ;;
  *)
    echo "Invalid argument: $1"
    echo "Usage: sudo $0 <cloudcore|edgecore>"
    exit 1
    ;;
  esac

  echo "Configuration completed."
}

main "$@"
