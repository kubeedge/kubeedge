#!/bin/bash -ex

# Copyright 2025 The KubeEdge Authors.
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

curpath=$PWD
echo $PWD
CONTAINER_RUNTIME=${CONTAINER_RUNTIME:-"containerd"}

# build mapper project
cd ${curpath}/staging/src/github.com/kubeedge/mapper-framework
make generate modbus nostream && echo "successfully generated mapper project"

# copy device driver and add go work
cd ${curpath} && git clone https://github.com/wbc6080/modbus.git
cp -r ${curpath}/modbus/driver/* ${curpath}/staging/src/github.com/kubeedge/modbus/driver/
cp -r ${curpath}/modbus/config.yaml ${curpath}/staging/src/github.com/kubeedge/modbus/
go work use ./staging/src/github.com/kubeedge/modbus

# build modbus mapper image
cd ${curpath}/staging/src/github.com/kubeedge/modbus
CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go && sed -i '/go build/d' Dockerfile_nostream
docker build -f Dockerfile_nostream -t modbus-e2e-mapper:v1.0.0 . && echo "successfully build test mapper image"

# import images to container-runtime
docker save -o modbus-mapper.tar modbus-e2e-mapper:v1.0.0

if [[ "${CONTAINER_RUNTIME}" = "cri-o" ]]; then
  # Use podman to import the mapper image and change it to the correct name
  sudo podman load -i modbus-mapper.tar && sudo podman tag localhost/v1.0.0:latest docker.io/library/modbus-e2e-mapper:v1.0.0 && echo "successfully import modbus mapper image to CRI-O"
elif [[ "${CONTAINER_RUNTIME}" = "isulad" ]]; then
  sudo isula load -i modbus-mapper.tar && echo "successfully import modbus mapper image to Isulad"
elif [[ "${CONTAINER_RUNTIME}" = "containerd" ]]; then
  sudo ctr -n k8s.io images import modbus-mapper.tar && echo "successfully import modbus mapper image to Containerd"
elif [[ "${CONTAINER_RUNTIME}" = "docker" ]]; then
  echo "no need to import modbus mapper image"
else
  echo "not supported container runtime ${CONTAINER_RUNTIME}"
  exit 1
fi