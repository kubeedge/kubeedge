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
rm -rf /home/src/Edge-SDV
mkdir -p /home/src/
cp -r test/Edge-SDV /home/src/

modulename=$1

cd ${curpath}/test/Edge-SDV/utils/

result=$( docker images -q kubeedge_test/dat )

if [[ -n "$result" ]]; then
    echo "kubeedge_test/dat:v1 image is exist"
else
    echo "kubeedge_test/dat:v1 image is not exist, hence building the image"
    docker build -t kubeedge_test/dat:v1 -f ./Dockerfile .
fi

PWD=${curpath}/test/Edge-SDV

rm -rf $PWD/modules/edgecore/$modulename/$modulename.test
# Specify the module name to compile in below command
docker run --rm  -v $PWD:$PWD -w $PWD/scripts kubeedge_test/dat:v1 bash -x buildcase.sh $modulename
rm $PWD/config.json

export MQTT_CLIENT=127.0.0.1
:> /tmp/testcase.log
bash ${PWD}/scripts/fast_test $modulename 2>&1 | tee /tmp/fast_test.log && cat /tmp/fast_test.log >> /tmp/testcase.log && :> /tmp/fast_test.log

grep  -e "Running Suite" -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | sed -r 's/\x1B\[([0-9];)?([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g' | sed -r 's/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g'
echo "SDV Automation Test Final Summary Report"
echo "==============================================="
echo "Total Number of Test cases = `grep "Ran " /tmp/testcase.log | awk '{sum+=$2} END {print sum}'`"
passed=`grep -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | awk '{print $3}' | sed -r "s/\x1B\[([0-9];)?([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g" | awk '{sum+=$1} END {print sum}'`
echo "Number of Test cases PASSED = $passed"
fail=`grep -e "SUCCESS\!" -e "FAIL\!" /tmp/testcase.log | awk '{print $6}' | sed -r "s/\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[mGK]//g" | awk '{sum+=$1} END {print sum}'`
echo "Number of Test cases FAILED = $fail"
echo "==================Result Summary======================="
