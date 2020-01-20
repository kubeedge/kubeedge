#!/usr/bin/env bash

# Copyright 2019 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

KUBEEDGE_ROOT=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )/..
ENABLE_DAEMON=${ENABLE_DAEMON:-false}
LOG_DIR=${LOG_DIR:-"/tmp"}

if [[ "${CLUSTER_NAME}x" == "x" ]];then
    CLUSTER_NAME="test"
fi

export CLUSTER_CONTEXT="--name ${CLUSTER_NAME}"

function check_prerequisites {
  check_kubectl
  check_kind
  verify_go_version
  verify_docker_installed
}

# spin up cluster with kind command
function kind_up_cluster {
  echo "Running kind: [kind create cluster ${CLUSTER_CONTEXT}]"
  kind create cluster ${CLUSTER_CONTEXT}
}

function uninstall_kubeedge {
  # kill the cloudcore
  [[ -n "${CLOUDCORE_PID-}" ]] && sudo kill "${CLOUDCORE_PID}" 2>/dev/null

  # kill the edgecore
  [[ -n "${EDGECORE_PID-}" ]] && sudo kill "${EDGECORE_PID}" 2>/dev/null
}

# clean up
function cleanup {
  echo "Cleaning up..."
  uninstall_kubeedge

  echo "Running kind: [kind delete cluster ${CLUSTER_CONTEXT}]"
  kind delete cluster ${CLUSTER_CONTEXT}
}

if [[ "${ENABLE_DAEMON}" = false ]]; then
  trap cleanup EXIT
fi

function create_device_crd {
  echo "creating the device crd..."
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/devices/devices_v1alpha1_device.yaml
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/devices/devices_v1alpha1_devicemodel.yaml
}

function create_objectsync_crd {
  echo "creating the objectsync crd..."
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/reliablesyncs/cluster_objectsync_v1alpha1.yaml
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/reliablesyncs/objectsync_v1alpha1.yaml
}

function build_cloudcore {
  # TODO: make the binary to _output dir
  echo "building the cloudcore..."
  make -C "${KUBEEDGE_ROOT}" WHAT="cloudcore"
  mkdir -p ${KUBEEDGE_ROOT}/_output/bin/cloud
  mv ${KUBEEDGE_ROOT}/cloud/cloudcore ${KUBEEDGE_ROOT}/_output/bin/cloud
  mkdir -p ${KUBEEDGE_ROOT}/_output/bin/cloud/conf
}

function build_edgecore {
  echo "building the edgecore..."
  make -C "${KUBEEDGE_ROOT}" WHAT="edgecore"
  mkdir -p ${KUBEEDGE_ROOT}/_output/bin/edge
  mv ${KUBEEDGE_ROOT}/edge/edgecore ${KUBEEDGE_ROOT}/_output/bin/edge
  mkdir -p ${KUBEEDGE_ROOT}/_output/bin/edge/conf
}

function generate_certs {
  # generate the certs used for cloud and edge communication
  echo "generating the certs used for cloud and edge communication..."
  ${KUBEEDGE_ROOT}/build/tools/certgen.sh genCertAndKey edge
}

function start_cloudcore {
  CLOUD_CONFIGFILE=${KUBEEDGE_ROOT}/_output/bin/cloud/conf/cloud.yaml
  CLOUD_BIN=${KUBEEDGE_ROOT}/_output/bin/cloud/cloudcore
  ${CLOUD_BIN} --minconfig >  ${CLOUD_CONFIGFILE}
  sed -i "s|kubeConfig: .*|kubeConfig: ${KUBECONFIG}|g" ${CLOUD_CONFIGFILE}

  CLOUDCORE_LOG=${LOG_DIR}/cloudcore.log
  echo "start cloudcore..."
  nohup ${CLOUD_BIN} --config=${CLOUD_CONFIGFILE} > "${CLOUDCORE_LOG}" 2>&1 &
  CLOUDCORE_PID=$!
}

function create_node {
  echo "create edge node..."
  kubectl apply -f ${KUBEEDGE_ROOT}/build/node.json
}

function start_edgecore {
  EDGE_CONFIGFILE=${KUBEEDGE_ROOT}/_output/bin/edge/conf/edge.yaml
  EDGE_BIN=${KUBEEDGE_ROOT}/_output/bin/edge/edgecore
  ${EDGE_BIN} --minconfig >  ${EDGE_CONFIGFILE}
  sed -i "s|hostnameOverride: .*|hostnameOverride: edge-node|g" ${EDGE_CONFIGFILE}
  EDGECORE_LOG=${LOG_DIR}/edgecore.log

  export CHECK_EDGECORE_ENVIRONMENT="false"
  echo "start edgecore..."
  nohup ${EDGE_BIN} --config=${EDGE_CONFIGFILE} > "${EDGECORE_LOG}" 2>&1 &
  EDGECORE_PID=$!
}

function check_control_plane_ready {
  echo "wait the control-plane ready..."
  kubectl wait --for=condition=Ready node/test-control-plane --timeout=60s
}

# Check if all processes are still running. Prints a warning once each time
# a process dies unexpectedly.
function healthcheck {
  if [[ -n "${CLOUDCORE_PID-}" ]] && ! sudo kill -0 "${CLOUDCORE_PID}" 2>/dev/null; then
    echo "CloudCore terminated unexpectedly, see ${CLOUDCORE_LOG}"
    CLOUDCORE_PID=
  fi
  if [[ -n "${EDGECORE_PID-}" ]] && ! sudo kill -0 "${EDGECORE_PID}" 2>/dev/null; then
    echo "EdgeCore terminated unexpectedly, see ${EDGECORE_LOG}"
    EDGECORE_PID=
  fi
}

source "${KUBEEDGE_ROOT}/hack/lib/install.sh"

check_prerequisites

# Stop right away if there's an error
set -e

build_cloudcore
build_edgecore

kind_up_cluster

export KUBECONFIG=$HOME/.kube/config

check_control_plane_ready

# edge side don't support kind cni now, delete kind cni plugin for workaround
kubectl delete daemonset kindnet -nkube-system

create_device_crd
create_objectsync_crd
generate_certs
start_cloudcore
create_node
start_edgecore

if [[ "${ENABLE_DAEMON}" = false ]]; then
    echo "Local KubeEdge cluster is running. Press Ctrl-C to shut it down."
  else
    echo "Local KubeEdge cluster is running."
  fi
echo "Logs:
  /tmp/cloudcore.log
  /tmp/edgecore.log

To start using your kubeedge, you can run:

  export PATH=$PATH:$GOPATH/bin
  export KUBECONFIG=$HOME/.kube/config
  kubectl get nodes
"

if [[ "${ENABLE_DAEMON}" = false ]]; then
  while true; do sleep 1; healthcheck; done
fi

