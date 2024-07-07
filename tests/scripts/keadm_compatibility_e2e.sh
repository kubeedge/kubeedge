#!/usr/bin/env bash

# Copyright 2023 The KubeEdge Authors.
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

CLOUD_EDGE_VERSION=${1:-"v1.15.1"}
KUBEEDGE_ROOT=$PWD

source "${KUBEEDGE_ROOT}/hack/lib/install.sh"
source "${KUBEEDGE_ROOT}/tests/scripts/keadm_common_e2e.sh"

function cleanup() {
  sudo pkill edgecore || true
  helm uninstall cloudcore -n kubeedge && kubectl delete ns kubeedge  || true
  kind delete cluster --name test
  sudo rm -rf /var/log/kubeedge /etc/kubeedge /etc/systemd/system/edgecore.service $E2E_DIR/e2e_keadm/e2e_keadm.test $E2E_DIR/config.json
}

function build_image() {
  cd $KUBEEDGE_ROOT
  make image WHAT=installation-package -f $KUBEEDGE_ROOT/Makefile
  docker save kubeedge/installation-package:$IMAGE_TAG > installation-package.tar
  sudo ctr -n=k8s.io image import installation-package.tar
  kind load docker-image docker.io/kubeedge/installation-package:$IMAGE_TAG --name test

  set +e
  docker rmi $(docker images -f "dangling=true" -q)
  set -Ee
}

function get_cloudcore_image() {
   docker pull kubeedge/cloudcore:$CLOUD_EDGE_VERSION
   docker tag kubeedge/cloudcore:$CLOUD_EDGE_VERSION docker.io/kubeedge/cloudcore:$CLOUD_EDGE_VERSION
   kind load docker-image docker.io/kubeedge/cloudcore:$CLOUD_EDGE_VERSION --name test

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
  /usr/local/bin/keadm init --advertise-address=$MASTER_IP --kubeedge-version $CLOUD_EDGE_VERSION --set cloudCore.service.enable=false --kube-config=$KUBECONFIG --force

  # ensure tokensecret is generated
  while true; do
      sleep 3
      kubectl get secret -nkubeedge 2>/dev/null | grep -q tokensecret && break
  done

  cd $KUBEEDGE_ROOT
  export TOKEN=$(sudo /usr/local/bin/keadm gettoken --kube-config=$KUBECONFIG)
  sudo systemctl set-environment CHECK_EDGECORE_ENVIRONMENT="false"
  sudo /usr/local/bin/keadm join --token=$TOKEN --cloudcore-ipport=$MASTER_IP:10000 --edgenode-name=edge-node --kubeedge-version=$CLOUD_EDGE_VERSION

  # ensure edgenode is ready
  while true; do
      sleep 3
      kubectl get node | grep edge-node | grep -q -w Ready && break
  done
}

set -Ee
trap cleanup EXIT

echo -e "\nBuilding ginkgo test cases..."
build_ginkgo

export KUBECONFIG=$HOME/.kube/config

echo -e "\nPreparing cluster..."
prepare_cluster

echo -e "\nBuilding keadm image..."
build_image

echo -e "\nGet cloudcore image..."
get_cloudcore_image

install_cni_plugins

echo -e "\nStarting kubeedge..."
start_kubeedge

echo -e "\nRunning test..."
run_test
