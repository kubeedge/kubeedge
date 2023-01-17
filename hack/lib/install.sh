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

# check if kubectl installed
function check_kubectl {
  echo "checking kubectl"
  command -v kubectl >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    echo "kubectl not installed, exiting."
    exit 1
  else
    echo -n "found kubectl, " && kubectl version --short --client
  fi
}

# check if kind installed
function check_kind {
  echo "checking kind"
  command -v kind >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    echo "installing kind ."
    GO111MODULE="on" go install sigs.k8s.io/kind@v0.12.0
    if [[ $? -ne 0 ]]; then
      echo "kind installed failed, exiting."
      exit 1
    fi

    # avoid modifing go.sum and go.mod when installing the kind
    git checkout -- go.mod go.sum

    export PATH=$PATH:$GOPATH/bin
  else
    echo -n "found kind, version: " && kind version
  fi
}

# check if golangci-lint installed
function check_golangci-lint {
  GOPATH="${GOPATH:-$(go env GOPATH)}"
  echo "checking golangci-lint"
  export PATH=$PATH:$GOPATH/bin
  expectedVersion="1.42.0"
  command -v golangci-lint >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    install_golangci-lint
  else
    version=$(golangci-lint version)
    if [[ $version =~ $expectedVersion ]]; then
      echo -n "found golangci-lint, version: " && golangci-lint version
    else
      echo "golangci-lint version not matched, now version is $version, begin to install new version $expectedVersion"
      install_golangci-lint
    fi
  fi
}

function install_golangci-lint {
  echo "installing golangci-lint ."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin v1.42.0
    if [[ $? -ne 0 ]]; then
      echo "golangci-lint installed failed, exiting."
      exit 1
    fi

    export PATH=$PATH:$GOPATH/bin
}

verify_containerd_installed(){
  # verify the containerd installed
  command -v containerd >/dev/null || {
    echo "must install the containerd first"
    exit 1
  }
}

verify_docker_installed(){
  # verify the docker installed
  command -v docker >/dev/null || {
    echo "must install the docker first"
    exit 1
  }
}

# install CNI plugins
function install_cni_plugins() {
  # install CNI plugins if not exist
  if [ ! -f "/opt/cni/bin/loopback" ]; then
    echo -e "install CNI plugins..."
    mkdir -p /opt/cni/bin
    wget https://github.com/containernetworking/plugins/releases/download/v1.1.1/cni-plugins-linux-amd64-v1.1.1.tgz
    tar Cxzvf /opt/cni/bin cni-plugins-linux-amd64-v1.1.1.tgz
  else
    echo "CNI plugins already installed and no need to install"
  fi
}
