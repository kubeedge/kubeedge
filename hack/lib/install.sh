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
    echo -n "found kubectl, " && kubectl version --client
  fi
}

# check if kind installed
function check_kind {
  echo "checking kind"
  command -v kind >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    echo "installing kind ."
    GO111MODULE="on" go install sigs.k8s.io/kind@v0.21.0
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
  expectedVersion="1.54.2"
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
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin v1.54.2
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
    return 1
  }
}

verify_cridockerd_installed() {
  # verify the cri-dockerd installed
  command -v cri-dockerd >/dev/null || {
    echo "must install the cri-dockerd first"
    return 1
  }
}

verify_crio_installed() {
  # verify the cri-o installed
  command -v crio >/dev/null || {
    echo "must install the cri-o first"
    return 1
  }
}

verify_isulad_installed() {
  # verify the isulad installed
  command -v isulad >/dev/null || {
    echo "must install the isulad first"
    return 1
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
    sudo rm -rf ${CNI_PKG}
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
  else
    echo "CNI plugins already installed and no need to install"
  fi
}

function install_docker() {
  CRIDOCKERD_VERSION="v0.3.8"
  sudo apt-get update
  sudo apt-get install \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
                 $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
  sudo apt-get update
  sudo apt-get install docker-ce docker-ce-cli containerd.io
  git clone https://github.com/Mirantis/cri-dockerd.git -b ${CRIDOCKERD_VERSION}
  (
    export GOWORK=off
    cd cri-dockerd
    make cri-dockerd
    sudo install -o root -g root -m 0755 cri-dockerd /usr/local/bin/cri-dockerd
    sudo install packaging/systemd/* /etc/systemd/system
    sudo sed -i -e 's,/usr/bin/cri-dockerd,/usr/local/bin/cri-dockerd,' /etc/systemd/system/cri-docker.service
  )
  sudo systemctl daemon-reload
  sudo systemctl enable --now cri-docker.socket
  sudo systemctl restart cri-docker
  sudo rm -rf cri-dockerd
}

function install_crio() {
  CRIO_VERSION="1.28.2"
  sudo rm -rf cri-o.amd64.v${CRIO_VERSION}.tar.gz && sudo rm -rf cri-o
  sudo wget https://storage.googleapis.com/cri-o/artifacts/cri-o.amd64.v${CRIO_VERSION}.tar.gz
  sudo tar -zxvf cri-o.amd64.v${CRIO_VERSION}.tar.gz
  sudo sed -i 's/\/usr\/bin\/env sh/!\/bin\/bash/' cri-o/install
  sudo sed -i 's/ExecStart=.*/ExecStart=\/usr\/local\/bin\/crio --selinux=false \\/' cri-o/contrib/crio.service
  cd cri-o
  sudo /bin/bash ./install
  sudo systemctl daemon-reload
  sudo systemctl enable --now crio
  sudo systemctl restart crio
  cd .. && sudo rm -rf cri-o.amd64.v${CRIO_VERSION}.tar.gz && sudo rm -rf cri-o
}

install_isulad() {
  # export LDFLAGS
  set +u
  export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig:$PKG_CONFIG_PATH
  export LD_LIBRARY_PATH=/usr/local/lib:/usr/lib:/lib/x86_64-linux-gnu/:$LD_LIBRARY_PATH
  set -u

  sudo sh -c "echo '/usr/local/lib' >>/etc/ld.so.conf"
  CURRENT_PATH=$(
    cd $(dirname $0)
    pwd
  )
  sudo apt-get update
  sudo apt-get install -y g++ libprotobuf-dev protobuf-compiler protobuf-compiler-grpc libgrpc++-dev libgrpc-dev libtool automake autoconf cmake make pkg-config libyajl-dev zlib1g-dev libselinux1-dev libseccomp-dev libcap-dev libsystemd-dev git libarchive-dev libcurl4-gnutls-dev openssl libdevmapper-dev python3 libtar0 libtar-dev libhttp-parser-dev libwebsockets-dev
  BUILD_DIR=/tmp/build_isulad

  sudo rm -rf $BUILD_DIR
  sudo mkdir -p $BUILD_DIR

  sudo git config --global --add safe.directory /tmp/build_isulad/lxc/lxc-4.0.3
  # build lxc
  cd $BUILD_DIR
  sudo git clone https://gitee.com/src-openeuler/lxc.git -b openEuler-22.03-LTS-Next
  cd lxc
  sudo ./apply-patches
  cd lxc-4.0.3
  sudo ./autogen.sh
  sudo ./configure
  sudo make CFLAGS="-Wno-error=strict-prototypes -Wno-error=old-style-definition" -j $(nproc)
  sudo make install CFLAGS="-Wno-error=strict-prototypes -Wno-error=old-style-definition"

  # build lcr
  cd $BUILD_DIR
  sudo git clone https://gitee.com/openeuler/lcr.git -b v2.1.4
  cd lcr
  sudo mkdir build
  cd build
  sudo cmake ..
  sudo make -j $(nproc)
  sudo make install

  # build and install clibcni
  cd $BUILD_DIR
  sudo git clone https://gitee.com/openeuler/clibcni.git -b v2.1.0
  cd clibcni
  sudo mkdir build
  cd build
  sudo cmake ..
  sudo make -j $(nproc)
  sudo make install

  # build and install iSulad
  cd $BUILD_DIR
  sudo git clone https://gitee.com/openeuler/iSulad.git -b v2.1.5
  cd iSulad
  sudo mkdir build
  cd build
  sudo cmake -DENABLE_CRI_API_V1=ON ..
  sudo make -j $(nproc)
  sudo make install

  sudo apt-get install -y jq
  sudo sed -i 's#/usr/bin/isulad#/usr/local/bin/isulad#g' ../src/contrib/init/isulad.service
  sudo sed -i 's#-/etc/sysconfig/iSulad#/etc/isulad/daemon.json#g' ../src/contrib/init/isulad.service
  TMP_FILE=/home/runner/tmp.json
  ISULAD_CONF_FILE=/etc/isulad/daemon.json
  sudo cat ${ISULAD_CONF_FILE} | sudo jq '.["websocket-server-listening-port"]=10355' >${TMP_FILE} && sudo mv -f ${TMP_FILE} ${ISULAD_CONF_FILE}
  sudo cat ${ISULAD_CONF_FILE} | sudo jq '.["cni-bin-dir"]="/opt/cni/bin"' >${TMP_FILE} && sudo mv -f ${TMP_FILE} ${ISULAD_CONF_FILE}
  sudo cat ${ISULAD_CONF_FILE} | sudo jq '.["cni-conf-dir"]="/etc/cni/net.d"' >${TMP_FILE} && sudo mv -f ${TMP_FILE} ${ISULAD_CONF_FILE}
  sudo cat ${ISULAD_CONF_FILE} | sudo jq '.["network-plugin"]="cni"' >${TMP_FILE} && sudo mv -f ${TMP_FILE} ${ISULAD_CONF_FILE}
  sudo cat ${ISULAD_CONF_FILE} | sudo jq '.["enable-cri-v1"]=true' >${TMP_FILE} && sudo mv -f ${TMP_FILE} ${ISULAD_CONF_FILE}
  sudo cat ${ISULAD_CONF_FILE} | sudo jq '.["pod-sandbox-image"]="kubeedge/pause:3.6"' >${TMP_FILE} && sudo mv -f ${TMP_FILE} ${ISULAD_CONF_FILE}
  sudo cat ${ISULAD_CONF_FILE} | sudo jq '.["registry-mirrors"]=["docker.io"]' >${TMP_FILE} && sudo mv -f ${TMP_FILE} ${ISULAD_CONF_FILE}
  sudo cat ${ISULAD_CONF_FILE} | sudo jq '.["insecure-registries"]=["k8s.gcr.io"]' >${TMP_FILE} && sudo mv -f ${TMP_FILE} ${ISULAD_CONF_FILE}
  sudo cat /etc/isulad/daemon.json

  sudo cp ../src/contrib/init/isulad.service /usr/lib/systemd/system/
  sudo ldconfig
  sudo systemctl daemon-reload
  sudo systemctl enable isulad
  sudo systemctl restart isulad
  cd $CURRENT_PATH
  # clean
  sudo rm -rf $BUILD_DIR
  sudo apt autoremove
}
