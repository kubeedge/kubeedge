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

which ginkgo &> /dev/null || (
    go get github.com/onsi/ginkgo/ginkgo
    sudo cp $GOPATH/bin/ginkgo /usr/local/bin/
)

bash ${curpath}/tests/e2e/scripts/cleanup.sh deployment
bash ${curpath}/tests/e2e/scripts/cleanup.sh edgesite
bash ${curpath}/tests/e2e/scripts/cleanup.sh device_crd

E2E_DIR=${curpath}/tests/e2e
sudo rm -rf ${E2E_DIR}/deployment/deployment.test
sudo rm -rf ${E2E_DIR}/device_crd/device_crd.test

# Specify the module name to compile in below command
bash -x ${E2E_DIR}/scripts/compile.sh $1

ENABLE_DAEMON=true bash -x ${curpath}/hack/local-up-kubeedge.sh

kubectl create clusterrolebinding system:anonymous --clusterrole=cluster-admin --user=system:anonymous

:> /tmp/testcase.log

bash -x ${E2E_DIR}/scripts/fast_test.sh $1

#stop the edgecore after the test completion
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
