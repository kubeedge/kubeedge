#!/usr/bin/env bash

# Copyright 2024 The KubeEdge Authors.
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
E2E_DIR=$(realpath $(dirname $0)/..)

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