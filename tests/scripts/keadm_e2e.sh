#!/usr/bin/env bash

# Copyright 2021 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

KUBEEDGE_ROOT=$PWD
IMAGE_TAG=$(git describe --tags 2>/dev/null || echo "v0.0.0")
KUBEEDGE_VERSION=$IMAGE_TAG

source "${KUBEEDGE_ROOT}/hack/lib/install.sh"
source "${KUBEEDGE_ROOT}/tests/scripts/keadm_common_e2e.sh"

function cleanup() {
  sudo pkill edgecore || true
  helm uninstall cloudcore -n kubeedge || true
  # The namespace cleanup timeout may occur if pods is not clearned
  kubectl get pods -n kubeedge | awk '{print $1}' | grep -v NAME | xargs kubectl delete pods -n kubeedge --force --grace-period=0 || true
  kubectl delete ns kubeedge --force --grace-period=0  || true
  kind delete cluster --name test
  sudo rm -rf /var/log/kubeedge /etc/kubeedge /etc/systemd/system/edgecore.service $E2E_DIR/e2e_keadm/e2e_keadm.test $E2E_DIR/config.json
}

function build_image() {
  cd $KUBEEDGE_ROOT
  make image WHAT=cloudcore -f $KUBEEDGE_ROOT/Makefile
  make image WHAT=installation-package -f $KUBEEDGE_ROOT/Makefile
  # convert docker images to cri image, or cri runtime cannot identify the image that already existed on the local host
  echo "save docker images to cri images"
  docker save kubeedge/cloudcore:$IMAGE_TAG > cloudcore.tar
  docker save kubeedge/installation-package:$IMAGE_TAG > installation-package.tar
  sudo ctr -n=k8s.io image import cloudcore.tar
  sudo ctr -n=k8s.io image import installation-package.tar
  # load image to test cluster
  kind load docker-image docker.io/kubeedge/cloudcore:$IMAGE_TAG --name test
  kind load docker-image docker.io/kubeedge/installation-package:$IMAGE_TAG --name test

  set +e
  docker rmi $(docker images -f "dangling=true" -q)
  docker system prune -f
  set -Ee
}

function start_kubeedge() {
  sudo mkdir -p /var/lib/kubeedge
  cd $KUBEEDGE_ROOT
  export MASTER_IP=`kubectl get node test-control-plane -o jsonpath={.status.addresses[0].address}`
  export KUBECONFIG=$HOME/.kube/config
  docker run --rm kubeedge/installation-package:$IMAGE_TAG cat /usr/local/bin/keadm > /usr/local/bin/keadm && chmod +x /usr/local/bin/keadm
  /usr/local/bin/keadm init --advertise-address=$MASTER_IP --kubeedge-version $KUBEEDGE_VERSION --set cloudCore.service.enable=false --kube-config=$KUBECONFIG --force
  
  # ensure tokensecret is generated
  while true; do
      sleep 3
      kubectl get secret -nkubeedge 2>/dev/null | grep -q tokensecret && break
  done
  
  cd $KUBEEDGE_ROOT
  export TOKEN=$(sudo /usr/local/bin/keadm gettoken --kube-config=$KUBECONFIG)
  sudo systemctl set-environment CHECK_EDGECORE_ENVIRONMENT="false"
  sudo -E CHECK_EDGECORE_ENVIRONMENT="false" /usr/local/bin/keadm join --token=$TOKEN --cloudcore-ipport=$MASTER_IP:10000 --edgenode-name=edge-node --kubeedge-version=$KUBEEDGE_VERSION

  # ensure edgenode is ready
  while true; do
      sleep 3
      kubectl get node | grep edge-node | grep -q -w Ready && break
  done
}

function check_edgecore_missing_kind_error() {
  local pattern="failed to unmarshal message content to unstructured obj:.*Object 'Kind' is missing"

  if ! command -v journalctl >/dev/null 2>&1; then
    echo "journalctl is not available, skip edgecore missing-kind log check"
    return 0
  fi

  if ! sudo -n true >/dev/null 2>&1; then
    echo "passwordless sudo is not available, skip edgecore missing-kind log check"
    return 0
  fi

  local logs
  logs=$(sudo -n journalctl -u edgecore --since "@${E2E_EDGECORE_LOG_SINCE}" --no-pager 2>/dev/null || true)
  if echo "$logs" | grep -E "$pattern" >/dev/null 2>&1; then
    echo "Found missing-kind decode error in edgecore logs:"
    echo "$logs" | grep -E "$pattern"
    return 1
  fi

  echo "No missing-kind decode error found in edgecore logs."
  return 0
}

set -Ee
trap cleanup EXIT
#trap cleanup ERR

echo -e "\nBuilding ginkgo test cases..."
build_ginkgo

export KUBECONFIG=$HOME/.kube/config

echo -e "\nPreparing cluster..."
prepare_cluster

echo -e "\nBuilding cloud image..."
build_image

install_cni_plugins

export E2E_EDGECORE_LOG_SINCE=$(date +%s)

echo -e "\nStarting kubeedge..."
start_kubeedge

echo -e "\nRunning test..."
set +e
run_test
test_rc=$?
check_edgecore_missing_kind_error
log_rc=$?
set -e

if [[ $test_rc != 0 || $log_rc != 0 ]]; then
  exit 1
fi
