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

cd `dirname $0`
workdir=`pwd`
cd $workdir

debugflag="-v 6 -alsologtostderr"

runtest=$1

export K8SMasterForKubeEdge=http://121.244.95.60:12436
export K8SMasterForProvisionEdgeNodes=http://121.244.95.60:12458
#setup env
cd ../
#Pre-configurations required for running the suite.
#Any new config addition required corresponding code changes.
cat >config.json<<END
{
        "image_url": ["nginx", "hello-world"],
        "k8smasterforkubeedge": "$K8SMasterForKubeEdge",
        "node_num": 500,
        "imagerepo": "kubeedge",
        "k8smasterforprovisionedgenodes": "$K8SMasterForProvisionEdgeNodes",
        "cloudimageurl": "pavan187/cloudcore:v2.2",
        "edgeimageurl": "pavan187/edgecore:v2.2",
        "namespace":"default",
        "controllerstubport": 54321,
        "protocol": "websocket"
}
END


if [ $# -eq 0 ]
  then
    #run testcase
    ./loadtest/loadtest.test $debugflag 2>&1 | tee /tmp/perf_test.log && cat /tmp/perf_test.log >> /tmp/performace_test.log && :> /tmp/perf_test.log
    ./nodedensity/nodedensity.test $debugflag 2>&1 | tee /tmp/perf_test.log && cat /tmp/perf_test.log >> /tmp/performace_test.log && :> /tmp/perf_test.log
    ./hubtest/hubtest.test $debugflag 2>&1 | tee /tmp/perf_test.log && cat /tmp/perf_test.log >> /tmp/performace_test.log && :> /tmp/perf_test.log
else
    ./$runtest/$runtest.test $debugflag 2>&1 | tee /tmp/perf_test.log && cat /tmp/perf_test.log >> /tmp/performace_test.log && :> /tmp/perf_test.log
fi
