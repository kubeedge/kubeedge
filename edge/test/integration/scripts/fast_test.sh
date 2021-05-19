#!/bin/bash

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


debugflag="-test.v -ginkgo.v"
compilemodule=$1
runtest=$2

KUBEEDGE_ROOT=$(realpath $(dirname "${BASH_SOURCE[0]}")/../../../..)
cd $(dirname $(dirname "${BASH_SOURCE[0]}"))

#Pre-configurations required for running the suite.
#Any new config addition required corresponding code changes.
cat >config.json<<END
{
        "mqttEndpoint":"tcp://$MQTT_SERVER:1884",
        "testManager": "http://127.0.0.1:12345",
        "edgedEndpoint": "http://127.0.0.1:10350",
        "image_url": ["nginx:latest", "redis:latest"],
        "nodeId": "edge-node"
}
END

if [[ $# -eq 0 ]]; then
    #run testcase
    export KUBEEDGE_ROOT=$KUBEEDGE_ROOT
    ./appdeployment/appdeployment.test $debugflag 2>&1 | tee -a /tmp/testcase.log
    ./device/device.test  $debugflag  2>&1 | tee -a /tmp/testcase.log
    ./metaserver/metaserver.test $debugflag  2>&1 | tee -a /tmp/testcase.log
else
    ./$compilemodule/$compilemodule.test $debugflag $runtest 2>&1 | tee -a /tmp/testcase.log
fi

