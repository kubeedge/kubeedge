#!/usr/bin/env bash

# Copyright 2020 The KubeEdge Authors.
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

curpath=$PWD

make all WHAT=keadm

cd `dirname $0`
workdir=`pwd`
cd $workdir

cd ../
echo $PWD

ginkgo build -r keadm/

kind create cluster --name test
kubectl create clusterrolebinding system:anonymous --clusterrole=cluster-admin --user=system:anonymous

# edge side don't support kind cni now, delete kind cni plugin for workaround
kubectl delete daemonset kindnet -nkube-system

export KUBECONFIG=$HOME/.kube/config
sudo mkdir -p /var/lib/kubeedge
sudo ${curpath}/_output/local/bin/keadm init --kube-config=$KUBECONFIG --advertise-address=127.0.0.1

export MASTER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' test-control-plane`

# TODO: Check the secret has been created
sleep 3

TOKEN=`sudo ${curpath}/_output/local/bin/keadm gettoken --kube-config=$KUBECONFIG`
echo $TOKEN
sudo CHECK_EDGECORE_ENVIRONMENT="false" ${curpath}/_output/local/bin/keadm join --token=$TOKEN --cloudcore-ipport=127.0.0.1:10000 --edgenode-name=edge-node

#Pre-configurations required for running the suite.
#Any new config addition required corresponding code changes.
cat >config.json<<END
{
        "image_url": ["nginx", "nginx"],
        "k8smasterforkubeedge":"https://$MASTER_IP:6443",
        "dockerhubusername":"user",
        "dockerhubpassword":"password",
        "mqttendpoint":"tcp://127.0.0.1:1884",
        "kubeconfigpath":"$KUBECONFIG"
}
END

# TODO: check edgenode running
sleep 10s

:> /tmp/testcase.log
./keadm/keadm.test $debugflag 2>&1 | tee -a /tmp/testcase.log

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
