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

kill_edge_core() {
   sudo pkill edge_core
    #kill the edge_core process if it exists.
    sleep 5s
    if pgrep edge_core >/dev/null
    then
        echo "Failed to kill edge_core process !!"
        exit 1
    else
        echo "edge_core is successfully killed !!"
    fi
}

kill_edgecontroller() {
    sudo pkill edge
    #kill the edgecontroller process if it exists.
    sleep 5s
    if pgrep edgecontroller >/dev/null
    then
        echo "Failed to kill the edgecontroller !!"
        exit 1
    else
        echo "edgecontroller is successfully killed !!"
    fi
}

cleanup_files(){
    workdir=$GOPATH/src/github.com/kubeedge/kubeedge
    cd $workdir

    sudo rm -rf cloud/edgecontroller
    sudo rm -rf cloud/tmp/
    sudo rm -rf edge/edge.db
    sudo rm -rf edge/edge_core
    sudo rm -rf edge/tmp/
    sudo rm -rf tests/e2e/config.json
    sudo rm -rf tests/e2e/kubeedge.crt
    sudo rm -rf tests/e2e/kubeedge.csr
    sudo rm -rf tests/e2e/kubeedge.key
    sudo rm -rf tests/e2e/rootCA.crt
    sudo rm -rf tests/e2e/rootCA.key
    sudo rm -rf tests/e2e/rootCA.srl
    sudo rm -rf tests/e2e/deployment/deployment.test
}

kill_edge_core
kill_edgecontroller
cleanup_files
