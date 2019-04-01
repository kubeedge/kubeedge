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

workdir=`pwd`
cd $workdir

curpath=$PWD
echo $PWD

if [ ! -d "/var/lib/edged" ]; then
  sudo mkdir /var/lib/edged && sudo chown $USER:$USER /var/lib/edged
fi

#run the edge_core and edgecontroller bin to run the E2E
make edge_core
make edgecontroller
sleep 2s
#Kill the process if it exists
sudo pkill edgecontroller
sudo pkill edge_core
sleep 2s
#check the process are killed successfully
if pgrep edgecontroller >/dev/null
then
    echo "edgecontroller process is still Running, Please kill edgecontroller!!"
    exit 1
else
    echo "edgecontroller process is killed"
fi
if pgrep edge_core >/dev/null
then
    echo "edge_core process is still Running, Please kill edge_core!!"
     exit 1
else
    echo "edge_core process is killed"
fi

PWD=${curpath}/tests/e2e
sudo rm -rf $PWD/deployment/deployment.test
go get github.com/onsi/ginkgo/ginkgo
sudo cp $GOPATH/bin/ginkgo /usr/bin/
# Specify the module name to compile in below command
bash -x $PWD/scripts/compile.sh $1
export MASTER_IP=121.244.95.60
:> /tmp/testcase.log
bash -x ${PWD}/scripts/fast_test.sh $1
#stop the edge_core after the test completion
grep  -e "Running Suite" -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | sed -r 's/\x1B\[([0-9];)?([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g' | sed -r 's/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g'
echo "Integration Test Final Summary Report"
echo "==============================================="
echo "Total Number of Test cases = `grep "Ran " /tmp/testcase.log | awk '{sum+=$2} END {print sum}'`"
passed=`grep -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | awk '{print $3}' | sed -r "s/\x1B\[([0-9];)?([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g" | awk '{sum+=$1} END {print sum}'`
echo "Number of Test cases PASSED = $passed"
fail=`grep -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | awk '{print $6}' | sed -r "s/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g" | awk '{sum+=$1} END {print sum}'`
echo "Number of Test cases FAILED = $fail"
echo "==================Result Summary======================="

if [ "$fail" != "0" ];then
    echo "Integration suite has failures, Please check !!"
    exit 1
else
    echo "Integration suite successfully passed all the tests !!"
    exit 0
fi
