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

KUBEEDGE_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/..
ENABLE_DAEMON=${ENABLE_DAEMON:-false}
LOG_DIR=${LOG_DIR:-"/tmp"}
LOG_LEVEL=${LOG_LEVEL:-2}
TIMEOUT=${TIMEOUT:-60}s
PROTOCOL=${PROTOCOL:-"WebSocket"}
CONTAINER_RUNTIME=${CONTAINER_RUNTIME:-"containerd"}
KIND_IMAGE=${1:-"kindest/node:v1.28.0"}
echo -e "The installation of the cni plugin will overwrite the cni config file. Use export CNI_CONF_OVERWRITE=false to disable it."

if [[ "${CLUSTER_NAME}x" == "x" ]]; then
  CLUSTER_NAME="test"
fi

export CLUSTER_CONTEXT="--name ${CLUSTER_NAME}"

function install_cr() {
  attempt_num=0
  max_attempts=5

  while [ $attempt_num -lt $max_attempts ]
  do
    if  [ $attempt_num -eq 3 ]; then
      echo "Download failed multiple times, try to change apt source ..."
      sudo sed -i 's@//.*archive.ubuntu.com@//mirrors.ustc.edu.cn@g' /etc/apt/sources.list
      sudo apt-get update
    fi
    if [[ "${CONTAINER_RUNTIME}" = "docker" ]]; then
      install_docker
      verify_docker_installed
      if [ $? -ne 0 ]; then
        # docker was not downloaded normally
        attempt_num=$[$attempt_num+1]
        echo "Install docker failed. Retrying..."
        continue
      fi
      verify_cridockerd_installed
      if [ $? -ne 0 ]; then
        # cri-dockerd was not downloaded normally
        attempt_num=$[$attempt_num+1]
        echo "Install cri-dockerd failed. Retrying..."
        continue
      fi
      echo "docker has been downloaded successfully"
      break

    elif [[ "${CONTAINER_RUNTIME}" = "cri-o" ]]; then
      install_crio
      verify_crio_installed
      if [ $? -ne 0 ]; then
        # crio was not downloaded normally
        attempt_num=$[$attempt_num+1]
        echo "Install cri-o failed. Retrying..."
        continue
      fi
      echo "cri-o has been downloaded successfully"
      break

    elif [[ "${CONTAINER_RUNTIME}" = "isulad" ]]; then
      install_isulad
      verify_isulad_installed
      if [ $? -ne 0 ]; then
        # isulad was not downloaded normally
        attempt_num=$[$attempt_num+1]
        echo "Install isulad failed. Retrying..."
        continue
      fi
      echo "isulad has been downloaded successfully"
      break
    else
      echo "No need to download container runtime"
      break
    fi
  done

  if [ $attempt_num -eq $max_attempts ]; then
    # all retries failed
    echo "Task failed after $max_attempts attempts."
    exit 1
  fi

}

function check_prerequisites {
  kubeedge::golang::verify_golang_version
  check_kubectl
  check_kind
  if [[ "${CONTAINER_RUNTIME}" = "docker" ]]; then
    # if we will use docker as edgecore container runtime, we need to verify whether docker already installed
    verify_docker_installed
    verify_cridockerd_installed
  elif [[ "${CONTAINER_RUNTIME}" = "cri-o" ]]; then
    # function to verify if cri-o is installed
    verify_crio_installed
  elif [[ "${CONTAINER_RUNTIME}" = "isulad" ]]; then
    # function to verify if isulad is installed
    verify_isulad_installed
  elif [[ "${CONTAINER_RUNTIME}" = "containerd" ]]; then
    # we will use containerd as cri runtime, so need to verify whether containerd already installed
    verify_containerd_installed
  else
    echo "not supported container runtime ${CONTAINER_RUNTIME}"
    exit 1
  fi
}

# spin up cluster with kind command
function kind_up_cluster {
  echo "Running kind: [kind create cluster ${CLUSTER_CONTEXT}]"
  kind create cluster ${CLUSTER_CONTEXT} --image ${KIND_IMAGE}
}

function uninstall_kubeedge {
  # kill the cloudcore
  [[ -n "${CLOUDCORE_PID-}" ]] && sudo kill "${CLOUDCORE_PID}" 2>/dev/null

  # kill the edgecore
  [[ -n "${EDGECORE_PID-}" ]] && sudo kill "${EDGECORE_PID}" 2>/dev/null

  # delete data
  sudo rm -rf /tmp/etc/kubeedge /tmp/var/lib/kubeedge

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
else
  trap cleanup ERR
  trap cleanup INT
fi

function create_device_crd {
  echo "creating the device crd..."
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/devices/devices_v1beta1_device.yaml
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/devices/devices_v1beta1_devicemodel.yaml
}

function create_objectsync_crd {
  echo "creating the objectsync crd..."
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/reliablesyncs/cluster_objectsync_v1alpha1.yaml
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/reliablesyncs/objectsync_v1alpha1.yaml
}

function create_rule_crd {
  echo "creating the rule crd..."
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/router/router_v1_rule.yaml
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/router/router_v1_ruleEndpoint.yaml
}

function create_operation_crd {
  echo "creating the operation crd..."
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/operations/operations_v1alpha1_nodeupgradejob.yaml
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/operations/operations_v1alpha1_imageprepulljob.yaml
}

function create_serviceaccountaccess_crd {
  echo "creating the saaccess crd..."
  kubectl apply -f ${KUBEEDGE_ROOT}/build/crds/policy/policy_v1alpha1_serviceaccountaccess.yaml
}

function build_cloudcore {
  echo "building the cloudcore..."
  make -C "${KUBEEDGE_ROOT}" WHAT="cloudcore"
}

function build_edgecore {
  echo "building the edgecore..."
  make -C "${KUBEEDGE_ROOT}" WHAT="edgecore"
}

function start_cloudcore {
  CLOUD_CONFIGFILE=${KUBEEDGE_ROOT}/_output/local/bin/cloudcore.yaml
  CLOUD_BIN=${KUBEEDGE_ROOT}/_output/local/bin/cloudcore
  ${CLOUD_BIN} --defaultconfig >${CLOUD_CONFIGFILE}
  sed -i '/cloudStream:/{n;s/false/true/;}' ${CLOUD_CONFIGFILE}
  if [[ "${PROTOCOL}" = "QUIC" ]]; then
    sed -i '/quic:/{n;N;s/false/true/;}' ${CLOUD_CONFIGFILE}
  fi

  # enable dynamic controller
  sed -i '/dynamicController:/{n;s/false/true/;}' ${CLOUD_CONFIGFILE}

  sed -i -e "s|kubeConfig: .*|kubeConfig: ${KUBECONFIG}|g" \
    -e "s|/var/lib/kubeedge/|/tmp&|g" \
    -e "s|tlsCAFile: .*|tlsCAFile: /etc/kubeedge/ca/cloudhub/rootCA.crt|g" \
    -e "s|tlsCAKeyFile: .*|tlsCAKeyFile: /etc/kubeedge/ca/cloudhub/rootCA.key|g" \
    -e "s|tlsCertFile: .*|tlsCertFile: /etc/kubeedge/certs/cloudhub/server.crt|g" \
    -e "s|tlsPrivateKeyFile: .*|tlsPrivateKeyFile: /etc/kubeedge/certs/cloudhub/server.key|g" \
    -e "s|/etc/|/tmp/etc/|g" \
    -e '/router:/{n;N;N;N;N;s/false/true/}' ${CLOUD_CONFIGFILE}
  CLOUDCORE_LOG=${LOG_DIR}/cloudcore.log
  echo "start cloudcore..."
  nohup sudo ${CLOUD_BIN} --config=${CLOUD_CONFIGFILE} --v=${LOG_LEVEL} >"${CLOUDCORE_LOG}" 2>&1 &
  CLOUDCORE_PID=$!

  # ensure tokensecret is generated
  while true; do
    sleep 3
    kubectl get secret -nkubeedge | grep -q tokensecret && break
  done
}

function start_edgecore {
  EDGE_CONFIGFILE=${KUBEEDGE_ROOT}/_output/local/bin/edgecore.yaml
  EDGE_BIN=${KUBEEDGE_ROOT}/_output/local/bin/edgecore
  ${EDGE_BIN} --defaultconfig >${EDGE_CONFIGFILE}

  sed -i '/edgeStream:/{n;s/false/true/;}' ${EDGE_CONFIGFILE}
  sed -i '/metaServer:/{n;s/false/true/;}' ${EDGE_CONFIGFILE}

  if [[ "${PROTOCOL}" = "QUIC" ]]; then
    sed -i '/quic:/{n;s/false/true/;}' ${EDGE_CONFIGFILE}
    sed -i '/websocket:/{n;s/true/false/;}' ${EDGE_CONFIGFILE}
  fi

  # if we will use docker as edgecore container runtime
  # we need to change edgecore container runtime from default containerd to docker
  if [[ "${CONTAINER_RUNTIME}" = "docker" ]]; then
    sed -i 's|imageServiceEndpoint: .*|imageServiceEndpoint: unix:///var/run/cri-dockerd.sock|' ${EDGE_CONFIGFILE}
    sed -i 's|containerRuntimeEndpoint: .*|containerRuntimeEndpoint: unix:///var/run/cri-dockerd.sock|' ${EDGE_CONFIGFILE}
  fi

  if [[ "${CONTAINER_RUNTIME}" = "cri-o" ]]; then
    sed -i 's|imageServiceEndpoint: .*|imageServiceEndpoint: unix:///var/run/crio/crio.sock|' ${EDGE_CONFIGFILE}
    sed -i 's|containerRuntimeEndpoint: .*|containerRuntimeEndpoint: unix:///var/run/crio/crio.sock|' ${EDGE_CONFIGFILE}
    sed -i 's|cgroupDriver: .*|cgroupDriver: systemd|' ${EDGE_CONFIGFILE}
  fi

  if [[ "${CONTAINER_RUNTIME}" = "isulad" ]]; then
    sed -i 's|imageServiceEndpoint: .*|imageServiceEndpoint: unix:///var/run/isulad.sock|' ${EDGE_CONFIGFILE}
    sed -i 's|containerRuntimeEndpoint: .*|containerRuntimeEndpoint: unix:///var/run/isulad.sock|' ${EDGE_CONFIGFILE}
  fi

  token=$(kubectl get secret -nkubeedge tokensecret -o=jsonpath='{.data.tokendata}' | base64 -d)

  sed -i -e "s|token: .*|token: ${token}|g" \
    -e "s|hostnameOverride: .*|hostnameOverride: edge-node|g" \
    -e "s|/etc/|/tmp/etc/|g" \
    -e "s|/var/lib/kubeedge/|/tmp&|g" \
    -e "s|mqttMode: .*|mqttMode: 0|g" \
    -e '/serviceBus:/{n;s/false/true/;}' ${EDGE_CONFIGFILE}

  sed -i -e "s|/tmp/etc/resolv|/etc/resolv|g" ${EDGE_CONFIGFILE}
  EDGECORE_LOG=${LOG_DIR}/edgecore.log

  echo "start edgecore..."
  export CHECK_EDGECORE_ENVIRONMENT="false"
  nohup sudo -E ${EDGE_BIN} --config=${EDGE_CONFIGFILE} --v=${LOG_LEVEL} >"${EDGECORE_LOG}" 2>&1 &
  EDGECORE_PID=$!
}

function check_control_plane_ready {
  echo "wait the control-plane ready..."
  kubectl wait --for=condition=Ready node/${CLUSTER_NAME}-control-plane --timeout=${TIMEOUT}
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

function generate_streamserver_cert {
  CA_PATH=${CA_PATH:-/tmp/etc/kubeedge/ca}
  CERT_PATH=${CERT_PATH:-/tmp/etc/kubeedge/certs}
  STREAM_KEY_FILE=${CERT_PATH}/stream.key
  STREAM_CSR_FILE=${CERT_PATH}/stream.csr
  STREAM_CRT_FILE=${CERT_PATH}/stream.crt
  K8SCA_FILE=/tmp/etc/kubernetes/pki/ca.crt
  K8SCA_KEY_FILE=/tmp/etc/kubernetes/pki/ca.key
  streamsubject=${SUBJECT:-/C=CN/ST=Zhejiang/L=Hangzhou/O=KubeEdge}

  if [[ ! -d /tmp/etc/kubernetes/pki ]]; then
    mkdir -p /tmp/etc/kubernetes/pki
  fi
  if [[ ! -d $CA_PATH ]]; then
    mkdir -p $CA_PATH
  fi
  if [[ ! -d $CERT_PATH ]]; then
    mkdir -p $CERT_PATH
  fi

  docker cp ${CLUSTER_NAME}-control-plane:/etc/kubernetes/pki/ca.crt $K8SCA_FILE
  docker cp ${CLUSTER_NAME}-control-plane:/etc/kubernetes/pki/ca.key $K8SCA_KEY_FILE
  cp /tmp/etc/kubernetes/pki/ca.crt /tmp/etc/kubeedge/ca/streamCA.crt

  SUBJECTALTNAME="subjectAltName = IP.1:127.0.0.1"
  echo $SUBJECTALTNAME >/tmp/server-extfile.cnf

  touch ~/.rnd

  openssl genrsa -out ${STREAM_KEY_FILE} 2048
  openssl req -new -key ${STREAM_KEY_FILE} -subj ${streamsubject} -out ${STREAM_CSR_FILE}
  openssl x509 -req -in ${STREAM_CSR_FILE} -CA ${K8SCA_FILE} -CAkey ${K8SCA_KEY_FILE} -CAcreateserial -out ${STREAM_CRT_FILE} -days 5000 -sha256 -extfile /tmp/server-extfile.cnf
}

cleanup

source "${KUBEEDGE_ROOT}/hack/lib/golang.sh"
source "${KUBEEDGE_ROOT}/hack/lib/install.sh"

install_cr

check_prerequisites

# Stop right away if there's an error
set -eE

build_cloudcore
build_edgecore

kind_up_cluster

export KUBECONFIG=$HOME/.kube/config

check_control_plane_ready

# edge side don't support kind cni now, delete kind cni plugin for workaround
kubectl delete daemonset kindnet -nkube-system
kubectl create ns kubeedge

create_device_crd
create_objectsync_crd
create_rule_crd
create_operation_crd
create_serviceaccountaccess_crd

generate_streamserver_cert

start_cloudcore

# install CNI plugins
if [[ "${CONTAINER_RUNTIME}" = "containerd" || "${CONTAINER_RUNTIME}" = "docker" || "${CONTAINER_RUNTIME}" = "isulad" ]]; then
  # we need to install CNI plugins only when we use remote(containerd) as edgecore container runtime
  install_cni_plugins
  if [[ "${CONTAINER_RUNTIME}" = "docker" ]]; then
    sudo systemctl restart docker
    sudo systemctl restart cri-docker
    sleep 2
  elif [[ "${CONTAINER_RUNTIME}" = "containerd" ]]; then
    sudo systemctl restart containerd
    sleep 2
  elif [[ "${CONTAINER_RUNTIME}" = "isulad" ]]; then
    sudo systemctl restart isulad
    sleep 2
  fi
fi

sleep 2

start_edgecore

if [[ "${ENABLE_DAEMON}" = false ]]; then
  echo "Local KubeEdge cluster is running. Press Ctrl-C to shut it down."
else
  echo "Local KubeEdge cluster is running. Use \"kill $BASHPID\" to shut it down."
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
  while true; do
    sleep 1
    healthcheck
  done
else
  while true; do
    sleep 3
    kubectl get nodes | grep edge-node | grep -q -w Ready && break
  done
  kubectl label node edge-node disktype=test
fi
