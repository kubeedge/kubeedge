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
WORKDIR=$(dirname $0)
E2E_DIR=$(realpath $(dirname $0)/..)
IMAGE_TAG=$(git describe --tags)
KUBEEDGE_VERSION=$IMAGE_TAG

source "${KUBEEDGE_ROOT}/hack/lib/install.sh"

function cleanup() {
  sudo pkill edgecore || true
  helm uninstall cloudcore -n kubeedge && kubectl delete ns kubeedge  || true
  kind delete cluster --name test
  sudo rm -rf /var/log/kubeedge /etc/kubeedge /etc/systemd/system/edgecore.service $E2E_DIR/e2e_keadm/e2e_keadm.test $E2E_DIR/config.json
}

function build_ginkgo() {
  cd $E2E_DIR
  ginkgo build -r e2e_keadm/
}

function prepare_cluster() {
  kind create cluster --name test

  echo "wait the control-plane ready..."
  kubectl wait --for=condition=Ready node/test-control-plane --timeout=60s

  kubectl create clusterrolebinding system:anonymous --clusterrole=cluster-admin --user=system:anonymous

  # edge side don't support kind cni now, delete kind cni plugin for workaround
  kubectl delete daemonset kindnet -nkube-system
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
}

function start_kubeedge() {
  sudo mkdir -p /var/lib/kubeedge
  cd $KUBEEDGE_ROOT
  export MASTER_IP=`kubectl get node test-control-plane -o jsonpath={.status.addresses[0].address}`
  export KUBECONFIG=$HOME/.kube/config
  docker run --rm kubeedge/installation-package:$IMAGE_TAG cat /usr/local/bin/keadm > /usr/local/bin/keadm && chmod +x /usr/local/bin/keadm

  /usr/local/bin/keadm init --advertise-address=$MASTER_IP --profile version=$KUBEEDGE_VERSION --set cloudCore.service.enable=false --kube-config=$KUBECONFIG --force
  
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

function run_test() {
  :> /tmp/testcase.log
  cd $E2E_DIR

  export ACK_GINKGO_RC=true

  ginkgo -v ./e2e_keadm/e2e_keadm.test -- \
  --image-url=nginx \
  --image-url=nginx \
  --kube-master="https://$MASTER_IP:6443" \
  --kubeconfig=$KUBECONFIG \
  --test.v
  if [[ $? != 0 ]]; then
      echo "Integration suite has failures, Please check !!"
      exit 1
  else
      echo "Integration suite successfully passed all the tests !!"
      exit 0
  fi
}

set -Ee
trap cleanup EXIT
trap cleanup ERR

echo -e "\nBuilding ginkgo test cases..."
build_ginkgo

export KUBECONFIG=$HOME/.kube/config

echo -e "\nPreparing cluster..."
prepare_cluster

echo -e "\nBuilding cloud image..."
build_image

install_cni_plugins

echo -e "\nStarting kubeedge..."
start_kubeedge

echo -e "\nRunning test..."
run_test
