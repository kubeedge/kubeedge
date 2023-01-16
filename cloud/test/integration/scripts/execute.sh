#!/bin/bash

# Copyright 2022 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd -P)"
TEST_DIR=$(dirname $(dirname "${BASH_SOURCE[0]}"))
APPS_CRD_DIR="${ROOT_DIR}/build/crds/apps"
ENVTEST_DOWNLOAD_DIR="/tmp/envtest/bin"
ENVTEST_BIN_DIR=""

function do_preparation() {
    which setup-envtest &> /dev/null || {
        go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
        sudo cp $GOPATH/bin/setup-envtest /usr/bin/
    }

    ENVTEST_BIN_DIR=$(setup-envtest use 1.22.1 --bin-dir=${ENVTEST_DOWNLOAD_DIR} -p path)

    which ginkgo &>/dev/null || {
        go install github.com/onsi/ginkgo/ginkgo@latest
        sudo cp $GOPATH/bin/ginkgo /usr/bin/
    }
}

function run_test() {
    modpkg="$(head -1 ${ROOT_DIR}/go.mod | awk '{print $2}')"/cloud/test/integration/controllermanager
    ldflags="-X ${modpkg}.appsCRDDirectoryPath=${APPS_CRD_DIR} \
             -X ${modpkg}.envtestBinDir=${ENVTEST_BIN_DIR}"

    ginkgo --ldflags "${ldflags}" -v ${TEST_DIR}/controllermanager
}

do_preparation

run_test
