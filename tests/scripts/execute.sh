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

set -e

workdir=`pwd`
cd $workdir

curpath=$PWD
echo $PWD

GOPATH=${GOPATH:-$(go env GOPATH)}

which ginkgo &> /dev/null || (
    go install github.com/onsi/ginkgo/ginkgo@latest
    sudo cp $GOPATH/bin/ginkgo /usr/local/bin/
)

cleanup() {
    bash ${curpath}/tests/scripts/cleanup.sh
}

cleanup

sudo rm -rf ${curpath}/tests/e2e/e2e.test
sudo rm -rf ${curpath}/tests/e2e_keadm/e2e_keadm.test

# Specify the module name to compile in below command
bash -x ${curpath}/tests/scripts/compile.sh $1

ENABLE_DAEMON=true bash -x ${curpath}/hack/local-up-kubeedge.sh || {
    echo "failed to start cluster !!!"
    exit 1
}

:> /tmp/testcase.log

export GINKGO_TESTING_RESULT=0

trap cleanup EXIT

bash -x ${curpath}/tests/scripts/fast_test.sh $1
