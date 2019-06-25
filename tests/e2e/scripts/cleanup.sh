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

kill_edgecore() {
   sudo pkill edgecore
    #kill the edgecore process if it exists.
    sleep 5s
    if pgrep edgecore >/dev/null
    then
        echo "Failed to kill edgecore process !!"
        exit 1
    else
        echo "edgecore is successfully killed !!"
    fi
}

kill_cloudcore() {
    sudo pkill cloudcore
    #kill the cloudcore process if it exists.
    sleep 5s
    if pgrep cloudcore >/dev/null
    then
        echo "Failed to kill the cloudcore !!"
    else
        echo "cloudcore is successfully killed !!"
    fi
}

kill_edgesite() {
    sudo pkill edgesite
    #kill the edgecore process if it exists.
    sleep 5s
    if pgrep edgesite >/dev/null
    then
        echo "Failed to kill edgesite process !!"
    else
        echo "edgesite is successfully killed !!"
    fi
}

cleanup_files(){
    workdir=$GOPATH/src/github.com/kubeedge/kubeedge
    cd $workdir

    sudo rm -rf cloud/cloudcore
    sudo rm -rf cloud/tmp/
    sudo rm -rf edge/edge.db
    sudo rm -rf edge/edgecore
    sudo rm -rf edge/tmp/
    sudo rm -rf tests/e2e/kubeedge.crt
    sudo rm -rf tests/e2e/kubeedge.csr
    sudo rm -rf tests/e2e/kubeedge.key
    sudo rm -rf tests/e2e/rootCA.crt
    sudo rm -rf tests/e2e/rootCA.key
    sudo rm -rf tests/e2e/rootCA.srl
}

if [ "edgecore" = ${setuptype} ]; then
    workdir=$GOPATH/src/github.com/kubeedge/kubeedge
    cd $workdir
    sudo rm -rf tests/e2e/edgecore/edgecore.test
fi

if [ "edgesite" = ${setuptype} ]; then
    workdir=$GOPATH/src/github.com/kubeedge/kubeedge
    cd $workdir
    sudo rm -rf tests/e2e/edgesite/edgesite.test
    sudo rm -rf tests/e2e/config.json

fi

if [ "pre_test" = ${setuptype} ]; then
    workdir=$GOPATH/src/github.com/kubeedge/kubeedge
    cd $workdir
    kill_edgesite
    kill_cloudcore
    kill_edgecore
    sudo rm -rf tests/e2e/edgesite/edgesite.test
    sudo rm -rf tests/e2e/config.json
    sudo rm -rf tests/e2e/edgecore/edgecore.test

fi

cleanup_files