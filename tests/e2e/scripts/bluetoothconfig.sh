#!/usr/bin/env bash
# Copyright 2019 The KubeEdge Authors.
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

dockerhubusername=$1
nodename=$2

SRC_DIR=${GOPATH}/src/github.com/kubeedge/kubeedge
BLUETOOTH_PATH=${SRC_DIR}/mappers/bluetooth_mapper/deployment.yaml

create_bluetooth_config() {
    if [ ! -f ${BLUETOOTH_PATH} ]; then
        echo "There is no deployment.yaml!"
        exit 1
    fi
    echo "file found !!!!!!!!!!!!!"
    sed -i "s|image: .*|image: ${dockerhubusername}/bluetooth_mapper:v1.0|g" ${BLUETOOTH_PATH}
    sed -i "30s|name: .*|name: device-profile-config-${nodename}|g" ${BLUETOOTH_PATH}
}

create_bluetooth_config
