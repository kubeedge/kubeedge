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

compilemodule=$1

#setup env
cd ../

export MASTER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' test-control-plane`
export KUBECONFIG=$HOME/.kube/config

export CHECK_EDGECORE_ENVIRONMENT="false"

if [ $# -eq 0 ]
then
    #run testcase
    ginkgo ./deployment/deployment.test -- \
    --image-url=nginx \
    --image-url=nginx \
    --kube-master="https://$MASTER_IP:6443" \
    --kubeconfig=$KUBECONFIG \
    2>&1 | tee -a /tmp/testcase.log
else
   ginkgo ./$compilemodule/$compilemodule.test \
    --image-url=nginx \
    --image-url=nginx \
    --kube-master="https://$MASTER_IP:6443" \
    --kubeconfig=$KUBECONFIG \
   2>&1 | tee -a  /tmp/testcase.log
fi
