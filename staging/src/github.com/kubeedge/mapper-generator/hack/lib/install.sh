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
    GO111MODULE="on" go get sigs.k8s.io/kind@v0.9.0
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
  echo "checking golangci-lint"
  command -v golangci-lint >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    echo "installing golangci-lint ."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.39.0
    if [[ $? -ne 0 ]]; then
      echo "golangci-lint installed failed, exiting."
      exit 1
    fi

    export PATH=$PATH:$GOPATH/bin
  else
    echo -n "found golangci-lint, version: " && golangci-lint version
  fi
}

verify_go_version(){
  if [[ -z "$(command -v go)" ]]; then
    echo "Can't find 'go' in PATH, please fix and retry.
See http://golang.org/doc/install for installation instructions."
    exit 1
  fi

  local go_version
  IFS=" " read -ra go_version <<< "$(go version)"
  local minimum_go_version
  minimum_go_version=go1.12.1
  if [[ "${minimum_go_version}" != $(echo -e "${minimum_go_version}\n${go_version[2]}" | sort -s -t. -k 1,1 -k 2,2n -k 3,3n | head -n1) && "${go_version[2]}" != "devel" ]]; then
    echo "Detected go version: ${go_version[*]}.
Kubernetes requires ${minimum_go_version} or greater.
Please install ${minimum_go_version} or later."
    exit 1
  fi
}

verify_docker_installed(){
  # verify the docker installed
  command -v docker >/dev/null || {
    echo "must install the docker first"
    exit 1
  }
}

