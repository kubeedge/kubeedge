#!/bin/bash
set -e

CILIUM_DS="cilium"
KUBEEDGE_NAMESPACE="kubeedge"
WORK_DIR="./kubeedge_work_tmp"

install_yq() {
  CPU_ARCH=$(dpkg --print-architecture)
  if ! command -v yq >/dev/null 2>&1; then
    # install yq
    curl -LJO https://github.com/mikefarah/yq/releases/latest/download/"yq_linux_${CPU_ARCH}"
    sudo mv "yq_linux_${CPU_ARCH}" /usr/local/bin/yq
    sudo chmod a+x /usr/local/bin/yq
  fi
}

check_prerequisites() {
  command -v kubectl >/dev/null 2>&1 || {
    echo >&2 "kubectl required but not found. Aborting."
    exit 1
  }

  if ! kubectl get ns -A | grep -q $KUBEEDGE_NAMESPACE; then
    echo "KubeEdge not detected. Aborting."
    exit 1
  fi

  if ! kubectl get ds -A | grep -q $CILIUM_DS; then
    echo "Cilium not found. Please install Cilium first."
    exit 1
  fi

  mkdir $WORK_DIR
  install_yq
}

patch_cilium_daemonset() {
  echo "Patching Cilium DaemonSet..."
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
  echo "Enabling dynamicController..."
  FILE_NAME="${WORK_DIR}/cloudcore-configmap.yaml"
  kubectl get cm -n kubeedge cloudcore -o yaml >"$FILE_NAME"
  yq e '.data."cloudcore.yaml" |= (. | fromyaml | .modules.dynamicController.enable = true | toyaml)' -i "$FILE_NAME"

  kubectl apply -f "${FILE_NAME}"
  kubectl delete pod -n kubeedge --selector=kubeedge=cloudcore
}

patch_cilium_rbac() {
  echo "Update cilium clusterrole "
  FILE_NAME="${WORK_DIR}"/cilium-clusterrole.yaml
  kubectl get clusterrole cilium -o yaml >"$FILE_NAME"
  yq e '(.rules[] | select(.apiGroups[] == "cilium.io" and .resources[] == "ciliumpodippools").verbs) |= (. + ["get"] | unique)' -i "$FILE_NAME"
  kubectl apply -f "$FILE_NAME"

  FILE_NAME="${WORK_DIR}"/cilium-clusterrolebinding.yaml
  kubectl get clusterrolebinding cilium -o yaml >"$FILE_NAME"
  yq e '.subjects += [{"kind": "ServiceAccount", "name": "cloudcore", "namespace": "kubeedge"}, {"kind": "ServiceAccount", "name": "cloudcore", "namespace": "default"}]' -i "$FILE_NAME"
  kubectl apply -f "$FILE_NAME"
}

update_edgecore_config() {
  echo "Updating edgecore config..."
  # # TODO: use https://github.com/kubeedge/kubeedge/pull/6049

  # PATCH_FILE=/etc/kubeedge/config/edgecore.yaml

  # sudo yq e '.modules.edgeStream.enable = true' -i "$PATCH_FILE"
  # sudo yq e '.modules.edged.tailoredKubeletConfig.clusterDNS = ["10.96.0.10"]' -i "$PATCH_FILE"
  # sudo yq e '.modules.metaManager.metaServer.enable = true' -i "$PATCH_FILE"
  # sudo yq e '.modules.serviceBus.enable = true' -i "$PATCH_FILE"

  # systemctl daemon-reload
  # systemctl restart edgecore
}

deploy_edge_daemonset() {
  echo "Deploying cilium-edgecore..."
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

cleanup() {
  rm -rf ${WORK_DIR}
}

main() {
  check_prerequisites
  patch_cilium_daemonset
  enable_dynamic_controller
  patch_cilium_rbac
  update_edgecore_config
  deploy_edge_daemonset
  cleanup
  echo "Configuration completed."
}

main
