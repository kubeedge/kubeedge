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
HELM_DIR=$KUBEEDGE_ROOT/build/helm/charts
E2E_DIR=$(realpath $(dirname $0)/..)
IMAGE_TAG=$(git describe --tags)

function cleanup() {
  sudo pkill edgecore || true
  helm uninstall cloudcore -n kubeedge && kubectl delete ns kubeedge  || true
  kind delete cluster --name test
  sudo rm -rf /var/log/kubeedge /etc/kubeedge /etc/systemd/system/edgecore.service $E2E_DIR/keadm/keadm.test $E2E_DIR/config.json
}

function build_keadm() {
  make all WHAT=keadm
  cd $E2E_DIR
  ginkgo build -r keadm/
}

function prepare_cluster() {
  kind create cluster --name test

  echo "wait the control-plane ready..."
  kubectl wait --for=condition=Ready node/test-control-plane --timeout=60s

  kubectl create clusterrolebinding system:anonymous --clusterrole=cluster-admin --user=system:anonymous

  # edge side don't support kind cni now, delete kind cni plugin for workaround
  # kubectl delete daemonset kindnet -nkube-system
}

function build_cloud_image() {
  cd $KUBEEDGE_ROOT
  make image WHAT=cloudcore -f $KUBEEDGE_ROOT/Makefile
  kind load docker-image kubeedge/cloudcore:$IMAGE_TAG --name test
}

function start_kubeedge() {
  sudo mkdir -p /var/lib/kubeedge
  cd $KUBEEDGE_ROOT
  export KUBECONFIG=$HOME/.kube/config

  cd $HELM_DIR
  SET_ARGS="--set cloudCore.modules.cloudHub.advertiseAddress[0]=$MASTER_IP --set cloudCore.image.tag=$IMAGE_TAG --set cloudCore.service.enable=false"
  helm upgrade --install --wait --timeout 30s cloudcore ./cloudcore --namespace kubeedge --create-namespace -f ./cloudcore/values.yaml $SET_ARGS
  export MASTER_IP=`kubectl get node test-control-plane -o jsonpath={.status.addresses[0].address}`

  # ensure tokensecret is generated
  while true; do
      sleep 3
      kubectl get secret -nkubeedge 2>/dev/null | grep -q tokensecret && break
  done
  
  cd $KUBEEDGE_ROOT
  export TOKEN=$(sudo _output/local/bin/keadm gettoken --kube-config=$KUBECONFIG)
  sudo systemctl set-environment CHECK_EDGECORE_ENVIRONMENT="false"
  sudo -E CHECK_EDGECORE_ENVIRONMENT="false" _output/local/bin/keadm join --token=$TOKEN --cloudcore-ipport=$MASTER_IP:10000 --edgenode-name=edge-node

  #Pre-configurations required for running the suite.
  #Any new config addition required corresponding code changes.
  cat > $E2E_DIR/config.json <<END
{
        "image_url": ["nginx", "nginx"],
        "k8smasterforkubeedge":"https://$MASTER_IP:6443",
        "dockerhubusername":"user",
        "dockerhubpassword":"password",
        "mqttendpoint":"tcp://127.0.0.1:1884",
        "kubeconfigpath":"$KUBECONFIG"
}
END

  # ensure edgenode is ready
  while true; do
      sleep 3
      kubectl get node | grep edge-node | grep -q Ready && break
  done
}

function run_test() {
  :> /tmp/testcase.log
  cd $E2E_DIR
  ./keadm/keadm.test $debugflag 2>&1 | tee -a /tmp/testcase.log

  #stop the edgecore after the test completion
  grep  -e "Running Suite" -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | sed -r 's/\x1B\[([0-9];)?([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g' | sed -r 's/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g'
  echo "Integration Test Final Summary Report"
  echo "======================================================="
  echo "Total Number of Test cases = `grep "Ran " /tmp/testcase.log | awk '{sum+=$2} END {print sum}'`"
  passed=`grep -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | awk '{print $3}' | sed -r "s/\x1B\[([0-9];)?([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g" | awk '{sum+=$1} END {print sum}'`
  echo "Number of Test cases PASSED = $passed"
  fail=`grep -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | awk '{print $6}' | sed -r "s/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g" | awk '{sum+=$1} END {print sum}'`
  echo "Number of Test cases FAILED = $fail"
  echo "==================Result Summary======================="

  if [ "$fail" != "0" ];then
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

echo -e "\nBuilding keadm..."
build_keadm

export KUBECONFIG=$HOME/.kube/config

echo -e "\nPreparing cluster..."
prepare_cluster

echo -e "\nBuilding cloud image..."
build_cloud_image

echo -e "\nStarting kubeedge..."
start_kubeedge

echo -e "\nRunning test..."
run_test
