#!/bin/bash -x

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

compilemodule=$1
runtest=$2

export MASTER_IP=121.244.95.60
#setup env
cd ../
#Pre-configurations required for running the suite.
#Any new config addition required corresponding code changes.
cat >config.json<<END
{
        "image_url": ["nginx", "hello-world"],
        "k8smasterforkubeedge":"http://$MASTER_IP:12418"
}
END

if [ $# -eq 0 ]
  then
    #run testcase
    ./deployment/deployment.test $debugflag 2>&1 | tee /tmp/fast_test.log && cat /tmp/fast_test.log >> /tmp/testcase.log && :> /tmp/fast_test.log
    ./edgesite/edgesite.test $debugflag 2>&1 | tee /tmp/fast_test.log && cat /tmp/fast_test.log >> /tmp/testcase.log && :> /tmp/fast_test.log
else
    ./$compilemodule/$compilemodule.test $debugflag $runtest 2>&1 | tee /tmp/fast_test.log && cat /tmp/fast_test.log >> /tmp/testcase.log && :> /tmp/fast_test.log
fi
