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

SRC_DIR=${GOPATH}/src/github.com/kubeedge/kubeedge
EDGE_PATH=${SRC_DIR}/tests/performance/assets/02-edgeconfigmap.yaml
CLOUD_PATH=${SRC_DIR}/tests/performance/assets/01-configmap.yaml

nodename=$2
Url=$3
configmapName=$4
nodelimit=$5


create_edge_config() {
    if [ ! -f ${EDGE_PATH} ]; then
        echo "There is no 03-configmap-edgenodeconf.yaml!"
        exit 1
    fi
    echo "file found !!!!!!!!!!!!!"
    sed -i "s|namespace: .*|namespace: default|g" ${EDGE_PATH}
    sed -i "s|name: .*|name: ${configmapName}|g" ${EDGE_PATH}
    sed -i "s|node-id: .*|node-id: ${nodename}|g" ${EDGE_PATH}
    sed -i "s|hostname-override: .*|hostname-override: ${nodename}|g" ${EDGE_PATH}
    if [[ ${Url} == *"wss"* ]]; then
        sed -i "20s|url: .*|url: ${Url}/e632aba927ea4ac2b575ec1603d56f10/${nodename}/events|g" ${EDGE_PATH}
        sed -i "s|protocol: .*|protocol: websocket|g" ${EDGE_PATH}
    else
        sed -i "28s|url: .*|url: ${Url}|g" ${EDGE_PATH}
        sed -i "s|protocol: .*|protocol: quic|g" ${EDGE_PATH}
    fi
}

create_cloud_config() {
    if [ ! -f ${CLOUD_PATH} ]; then
        echo "There is no 01-configmap.yaml!"
        exit 1
    fi
    echo "file found !!!!!!!!!!!!!"
    sed -i "s|master: .*|master: ${Url}|g" ${CLOUD_PATH}
    sed -i "s|name: .*|name: ${configmapName}|g" ${CLOUD_PATH}
    sed -i "s|node-limit: .*|node-limit: ${nodelimit}|g" ${CLOUD_PATH}
}

"$@"