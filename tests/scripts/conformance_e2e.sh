#!/bin/bash

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

set -e
set -x

KUBEEDGE_ROOT=$PWD
TEST_DIR=$(realpath $(dirname $0)/..)

GOPATH=${GOPATH:-$(go env GOPATH)}
KIND_IMAGE=${1:-"kindest/node:v1.26.0"}
VERSION=$(git rev-parse --short=12 HEAD)

function cleanup() {
  bash ${KUBEEDGE_ROOT}/tests/scripts/cleanup.sh
}

function validate_ip() {
  local ip=$1
  if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
    return 0
  else
    return 1
  fi
}

cleanup

ENABLE_DAEMON=true bash -x ${KUBEEDGE_ROOT}/hack/local-up-kubeedge.sh ${KIND_IMAGE} || {
  echo "failed to start cluster !!!"
  exit 1
}

trap cleanup EXIT

sleep 10

MASTER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' test-control-plane`
if [ -z "$MASTER_IP" ] || ! validate_ip "$MASTER_IP"; then
  echo "error when get master ip: $MASTER_IP"
  exit 1
fi

if [ ! -f "$HOME/.kube/config" ]; then
  echo "not found kubeconfig file"
  exit 1
fi

export KUBECONFIG=$HOME/.kube/config

if [ ! -d "/tmp/results" ]; then
  mkdir -p /tmp/results
fi

rm -rf /tmp/results/*

function run_conformance_test() {
  local image_name=$1
  local tag_name=$2

  docker build -t "$image_name:$tag_name" -f ${KUBEEDGE_ROOT}/build/conformance/Dockerfile .

  docker run --rm \
  --env CHECK_EDGECORE_ENVIRONMENT="false" \
  --env ACK_GINKGO_RC="true" \
  --env KUBECONFIG=/root/.kube/config \
  --env RESULTS_DIR=/tmp/results \
  --env E2E_EXTRA_ARGS="--kube-master=https://${MASTER_IP}:6443" \
  -v ${KUBECONFIG}:/root/.kube/config \
  -v /tmp/results:/tmp/results \
  --network host "$image_name:$tag_name"
}

run_conformance_test "kubeedge/conformance-test" ${VERSION} || { echo "Conformance test failed with exit code $?"; exit 1; }