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

if [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) <tests>

  <tests>        the performance tests will be performed, it could be one of [loadtest, nodedensity, hubtest],
                 empty means all the performance tests will be performed.
Examples:
  $(basename $0)
  $(basename $0) loadtest
  $(basename $0) nodedensity
  $(basename $0) hubtest

EOF
  exit 0
fi

echo $PWD
curpath=$PWD
PWD=${curpath}/tests/performance
sudo rm -rf $PWD/loadtest/loadtest.test
sudo rm -rf $PWD/nodedensity/nodedensity.test
sudo rm -rf $PWD/hubtest/hubtest.test

go get github.com/onsi/ginkgo/ginkgo
sudo cp $GOPATH/bin/ginkgo /usr/bin/
# Specify the module name to compile in below command
bash -x $PWD/scripts/compileperf.sh $1

:> /tmp/testcase.log
bash -x ${PWD}/scripts/runperf.sh $1
#stop the edge_core after the test completion
grep  -e "Running Suite" -e "SUCCESS\!" -e "FAIL\!" /tmp/performace_test.log | sed -r 's/\x1B\[([0-9];)?([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g' | sed -r 's/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g'
echo "Performance Test Final Summary Report"
echo "==============================================="
echo "Total Number of Test cases = `grep "Ran " /tmp/loadtest_perf.log | awk '{sum+=$2} END {print sum}'`"
passed=`grep -e "SUCCESS\!" -e "FAIL\!" /tmp/performace_test.log | awk '{print $3}' | sed -r "s/\x1B\[([0-9];)?([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g" | awk '{sum+=$1} END {print sum}'`
echo "Number of Test cases PASSED = $passed"
fail=`grep -e "SUCCESS\!" -e "FAIL\!" /tmp/performace_test.log | awk '{print $6}' | sed -r "s/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g" | awk '{sum+=$1} END {print sum}'`
echo "Number of Test cases FAILED = $fail"
echo "==================Result Summary======================="

if [ "$fail" != "0" ];then
    echo "Performance tests has failures, Please check !!"
    exit 1
else
    echo "Performance tests successfully passed all the tests !!"
    exit 0
fi
