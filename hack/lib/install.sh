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
    GO111MODULE="on" go install sigs.k8s.io/kind@v0.18.0
    if [[ $? -ne 0 ]]; then
      echo "kind installed failed, exiting."
      exit 1
    fi

    # avoid modifying go.sum and go.mod when installing the kind
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
  expectedVersion="1.51.1"
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
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin v1.51.1
  if [[ $? -ne 0 ]]; then
    echo "golangci-lint installed failed, exiting."
    exit 1
  fi

  export PATH=$PATH:$GOPATH/bin
}

verify_containerd_installed() {
  # verify the containerd installed
  command -v containerd >/dev/null || {
    echo "must install the containerd first"
    exit 1
  }
}

verify_docker_installed() {
  # verify the docker installed
  command -v docker >/dev/null || {
    echo "must install the docker first"
    exit 1
  }
}

# install CNI plugins
function install_cni_plugins() {
  CNI_DOWNLOAD_ADDR=${CNI_DOWNLOAD_ADDR:-"https://github.com/containernetworking/plugins/releases/download/v1.1.1/cni-plugins-linux-amd64-v1.1.1.tgz"}
  CNI_PKG=${CNI_DOWNLOAD_ADDR##*/}
  CNI_CONF_OVERWRITE=${CNI_CONF_OVERWRITE:-"true"}
  echo -e "The installation of the cni plugin will overwrite the cni config file. Use export CNI_CONF_OVERWRITE=false to disable it."

  # install CNI plugins if not exist
  if [ ! -f "/opt/cni/bin/loopback" ]; then
    echo -e "start installing CNI plugins..."
    sudo mkdir -p /opt/cni/bin
    wget ${CNI_DOWNLOAD_ADDR}
    if [ ! -f ${CNI_PKG} ]; then
      echo -e "cni plugins package does not exits"
      exit 1
    fi
    sudo tar Cxzvf /opt/cni/bin ${CNI_PKG}
    rm -rf ${CNI_PKG}
    if [ ! -f "/opt/cni/bin/loopback" ]; then
      echo -e "the ${CNI_PKG} package does not contain a loopback file."
      exit 1
    fi

    # create CNI netconf file
    CNI_CONFIG_FILE="/etc/cni/net.d/10-containerd-net.conflist"
    if [ -f ${CNI_CONFIG_FILE} ]; then
      if [ ${CNI_CONF_OVERWRITE} == "false" ]; then
        echo -e "CNI netconf file already exist and will no overwrite"
        return
      fi
      echo -e "Configuring cni, ${CNI_CONFIG_FILE} already exists, will be backup as ${CNI_CONFIG_FILE}-bak ..."
      sudo mv ${CNI_CONFIG_FILE} ${CNI_CONFIG_FILE}-bak
    fi
    sudo mkdir -p "/etc/cni/net.d/"
    sudo sh -c 'cat > '${CNI_CONFIG_FILE}' <<EOF
{
  "cniVersion": "1.0.0",
  "name": "containerd-net",
  "plugins": [
    {
      "type": "bridge",
      "bridge": "cni0",
      "isGateway": true,
      "ipMasq": true,
      "promiscMode": true,
      "ipam": {
        "type": "host-local",
        "ranges": [
          [{
            "subnet": "10.88.0.0/16"
          }],
          [{
            "subnet": "2001:db8:4860::/64"
          }]
        ],
        "routes": [
          { "dst": "0.0.0.0/0" },
          { "dst": "::/0" }
        ]
      }
    },
    {
      "type": "portmap",
      "capabilities": {"portMappings": true}
    }
  ]
}
EOF'
    sudo systemctl restart containerd
    sleep 2
  else
    echo "CNI plugins already installed and no need to install"
  fi
}
