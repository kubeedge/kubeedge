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
TOOL_CACHE_DIR="${ROOT_DIR}/.cache/tools"
ENVTEST_BIN_DIR=""

SETUP_ENVTEST_VERSION="release-0.19"
GINKGO_VERSION="v2.21.0"

# Bin dirs are keyed by tool version so bumping SETUP_ENVTEST_VERSION or
# GINKGO_VERSION always triggers a fresh install instead of silently
# reusing a stale cached binary from a previous version.
SETUP_ENVTEST_BIN_DIR="${TOOL_CACHE_DIR}/bin/setup-envtest-${SETUP_ENVTEST_VERSION}"
GINKGO_BIN_DIR="${TOOL_CACHE_DIR}/bin/ginkgo-${GINKGO_VERSION}"
ENVTEST_DOWNLOAD_DIR="${TOOL_CACHE_DIR}/envtest/${SETUP_ENVTEST_VERSION}/bin"

function do_preparation() {
    mkdir -p "${SETUP_ENVTEST_BIN_DIR}" "${GINKGO_BIN_DIR}"

    [ -x "${SETUP_ENVTEST_BIN_DIR}/setup-envtest" ] || {
        GOBIN="${SETUP_ENVTEST_BIN_DIR}" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@${SETUP_ENVTEST_VERSION}
    }

    ENVTEST_BIN_DIR=$("${SETUP_ENVTEST_BIN_DIR}/setup-envtest" use 1.29.0 --bin-dir=${ENVTEST_DOWNLOAD_DIR} -p path)

    [ -x "${GINKGO_BIN_DIR}/ginkgo" ] || {
        GOBIN="${GINKGO_BIN_DIR}" go install github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION}
    }

    export PATH="${GINKGO_BIN_DIR}:${SETUP_ENVTEST_BIN_DIR}:${PATH}"
}

function run_test() {
    modpkg="$(head -1 ${ROOT_DIR}/go.mod | awk '{print $2}')"/cloud/test/integration/controllermanager
    ldflags="-X ${modpkg}.appsCRDDirectoryPath=${APPS_CRD_DIR} \
             -X ${modpkg}.envtestBinDir=${ENVTEST_BIN_DIR}"

    ginkgo --ldflags "${ldflags}" -v ${TEST_DIR}/controllermanager
}

do_preparation

run_test
