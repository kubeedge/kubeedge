#!/usr/bin/env bash

# Copyright 2020 The KubeEdge Authors.
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
KUBEEDGE_VERSION=${VERSION: 1}

source "${KUBEEDGE_ROOT}/hack/lib/golang.sh"
source "${KUBEEDGE_ROOT}/hack/lib/install.sh"
source "${KUBEEDGE_ROOT}/hack/lib/util.sh"

export KUBECONFIG=$HOME/.kube/config

kubeedge::version::get_version_info
VERSION=${GIT_VERSION}

function cleanup() {
  sudo pkill edgecore || true
  sudo systemctl stop edgecore.service && systemctl disable edgecore.service && rm /etc/systemd/system/edgecore.service || true
  sudo rm -rf /var/lib/kubeedge || true
  sudo pkill cloudcore || true
  kind delete cluster --name test
  sudo rm -rf /var/log/kubeedge /etc/kubeedge /etc/systemd/system/edgecore.service $E2E_DIR/e2e_keadm/e2e_keadm.test $E2E_DIR/config.json
  sudo rm -rf ${KUBEEDGE_ROOT}/_output/release/${VERSION}/
}

function build_keadm() {
  cd $KUBEEDGE_ROOT
  make all WHAT=keadm
  cd $E2E_DIR
  ginkgo build -r e2e_keadm/
}

function start_kubeedge() {
  sudo mkdir -p /var/lib/kubeedge
  cd $KUBEEDGE_ROOT

  sudo -E _output/local/bin/keadm deprecated init --kube-config=$KUBECONFIG --advertise-address=127.0.0.1 --kubeedge-version=${KUBEEDGE_VERSION}
  export MASTER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' test-control-plane`

  # ensure tokensecret is generated
  while true; do
      sleep 3
      kubectl get secret -nkubeedge 2>/dev/null | grep -q tokensecret && break
  done

  export TOKEN=$(sudo _output/local/bin/keadm gettoken --kube-config=$KUBECONFIG)
  sudo systemctl set-environment CHECK_EDGECORE_ENVIRONMENT="false"
  sudo _output/local/bin/keadm deprecated join --token=$TOKEN --cloudcore-ipport=127.0.0.1:10000 --edgenode-name=edge-node --kubeedge-version=${KUBEEDGE_VERSION}

  # ensure edgenode is ready
  while true; do
      sleep 3
      kubectl get node | grep edge-node | grep -q -w Ready && break
  done
}

trap cleanup EXIT
trap cleanup ERR

echo -e "\nUsing latest commit code to do keadm_deprecated_e2e test..."

echo -e "\nBuilding keadm..."
build_keadm

echo -e "\nPreparing cluster..."
util::prepare_cluster

install_cni_plugins

#if we use the local release version compiled with the latest codes, we need to copy release file and checksum file.
sudo mkdir -p /etc/kubeedge

sudo cp ${KUBEEDGE_ROOT}/_output/release/${VERSION}/kubeedge-${VERSION}-linux-amd64.tar.gz ${KUBEEDGE_ROOT}/_output/release/${VERSION}/checksum_kubeedge-${VERSION}-linux-amd64.tar.gz.txt /etc/kubeedge

echo -e "\nStarting kubeedge..." ${kubeedge_version}
start_kubeedge ${kubeedge_version}

echo -e "\nRunning test..."
run_test

# clean the before test
cleanup

echo -e "\nUsing latest official release version to do keadm_deprecated_e2e test..."

echo -e "\nBuilding keadm..."
build_keadm

echo -e "\nPreparing cluster..."
util::prepare_cluster

echo -e "\nStarting kubeedge..."
start_kubeedge ""

echo -e "\nRunning test..."
util::run_keadm_test
