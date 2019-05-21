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

setuptype=$1
nodename=$2
MASTER_IP=$3

SRC_DIR=${GOPATH}/src/github.com/kubeedge/kubeedge

echo `pwd`
CURRENT_PATH=`pwd`
EDGE_PATH=${SRC_DIR}/edge/conf/edge.yaml
CLOUD_PATH=${SRC_DIR}/cloud/conf/controller.yaml
EDGESITE_YAML_PATH=${SRC_DIR}/edgesite/conf/edgeSite.yaml

create_edge_config() {
    if [ ! -f ${EDGE_PATH} ]; then
        echo "There is no edge.yaml!"
        exit 1
    fi
    sed -i "s|certfile: .*|certfile: tmp/kubeedge.crt|g" ${EDGE_PATH}
    sed -i "s|keyfile: .*|keyfile: tmp/kubeedge.key|g" ${EDGE_PATH}
    sed -i "s|node-id: .*|node-id: ${nodename}|g" ${EDGE_PATH}
    sed -i "s|hostname-override: .*|hostname-override: ${nodename}|g" ${EDGE_PATH}
    sed -i "s|url: .*|url: wss://0.0.0.0:10000/e632aba927ea4ac2b575ec1603d56f10/${nodename}/events|g" ${EDGE_PATH}
    sed -i "s|mode: .*|mode: 0|g" ${EDGE_PATH}
}

create_cloud_config() {
    if [ ! -f ${CLOUD_PATH} ]; then
        echo "There is no controller.yaml!"
        exit 1
    fi
    sed -i "s|master: .*|master: ${MASTER_IP}|g" ${CLOUD_PATH}
    sed -i "s|ca: .*|ca: tmp/rootCA.crt|g" ${CLOUD_PATH}
    sed -i "s|cert: .*|cert: tmp/kubeedge.crt|g" ${CLOUD_PATH}
    sed -i "s|key: .*|key: tmp/kubeedge.key|g" ${CLOUD_PATH}
}

create_edgesite_config() {
    if [ ! -f ${EDGESITE_YAML_PATH} ]; then
        echo "There is no edgeSite.yaml!"
        exit 1
    fi
    sed -i "s|node-id: .*|node-id: ${nodename}|g" ${EDGESITE_YAML_PATH}
    sed -i "s|node-name: .*|node-name: ${nodename}|g" ${EDGESITE_YAML_PATH}
    sed -i "s|hostname-override: .*|hostname-override: ${nodename}|g" ${EDGESITE_YAML_PATH}
    sed -i "s|master: .*|master: ${MASTER_IP}|g" ${EDGESITE_YAML_PATH}
}

if [ "deployment" = ${setuptype} ]; then
    create_edge_config
    create_cloud_config
fi

if [ "edgesite" = ${setuptype} ]; then
    create_edgesite_config
fi
