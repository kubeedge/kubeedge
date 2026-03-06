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

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
SUDO=()
source "${SCRIPT_DIR}/privileged_runner.sh"
configure_privileged_runner "certificate generation"

if ! command -v openssl >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then
    "${SUDO[@]}" apt-get update
    "${SUDO[@]}" apt-get install openssl -y
  else
    echo "openssl is required but was not found"
    exit 1
  fi
fi

#Required
commonname=kubeedge

#Change to your company details
country=IN
state=Karnataka
locality=Bangalore
organization=Htipl
organizationalunit=IT
email=administrator@htipl.com

#Optional
password=root

echo "Generating key request for kubeedge"

"${SUDO[@]}" touch ~/.rnd
# Generate Root Key
"${SUDO[@]}" openssl genrsa -des3 -passout pass:$password -out rootCA.key 4096
# Generate Root Certificate
"${SUDO[@]}" openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.crt -passin pass:$password \
    -subj "/C=$country/ST=$state/L=$locality/O=$organization/OU=$organizationalunit/CN=$commonname/emailAddress=$email"
# Generate Key
"${SUDO[@]}" openssl genrsa -out kubeedge.key 2048
# Generate csr, Fill required details after running the command
"${SUDO[@]}" openssl req -new -key kubeedge.key -out kubeedge.csr -passin pass:$password \
    -subj "/C=$country/ST=$state/L=$locality/O=$organization/OU=$organizationalunit/CN=system:node:edge-node/emailAddress=$email"
# Generate Certificate
"${SUDO[@]}" openssl x509 -req -in kubeedge.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out kubeedge.crt -days 500 -sha256 -passin pass:$password

echo "---------------------------------------------"
echo "-----Certificate Generation is Completed-----"
echo "---------------------------------------------"

#Generate temparory folder in Edge and Cloud folders to copy certs
gopath_dir=$(go env GOPATH)
"${SUDO[@]}" mkdir -p $gopath_dir/src/github.com/kubeedge/kubeedge/edge/tmp
"${SUDO[@]}" mkdir -p $gopath_dir/src/github.com/kubeedge/kubeedge/cloud/tmp
#Copy the generated certs to respective paths
"${SUDO[@]}" mkdir -p /tmp/edgecore/
"${SUDO[@]}" cp -r rootCA.crt rootCA.key kubeedge.crt kubeedge.key /tmp/edgecore

echo "-----Certificate are Copied to Edge and Cloud Nodes-----"
