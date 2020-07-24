#!/usr/bin/env bash

# Copyright 2019 The KubeEdge Authors.
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
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/../../ && pwd)
export KUBE_CONFIG=""
export KUBE_APISERVER_URL="http://localhost:8080"
export TESTNS="crd-test"

runTests() {
  ret=0
  kubectl get namespace crd-test --no-headers --output=go-template={{.metadata.name}} > /dev/null 2>&1 || ret=$?
  if [[ "${ret}" -ne 0 ]]; then
    logStatus "creating test namespace for crds"
    kubectl create namespace ${TESTNS}
  fi

  ret=0
  kubectl get crd devicemodels.devices.kubeedge.io --no-headers --output=go-template={{.metadata.name}} > /dev/null 2>&1 || ret=$?
  if [[ "${ret}" -ne 0 ]]; then
    logStatus "Creating device model crd"
    kubectl create -f "${SCRIPT_ROOT}/build/crds/devices/devices_v1alpha2_devicemodel.yaml"
  fi

  ret=0
  kubectl get crd devices.devices.kubeedge.io --no-headers --output=go-template={{.metadata.name}} > /dev/null 2>&1 || ret=$?
  if [[ "${ret}" -ne 0 ]]; then
    logStatus "Creating device crd"
    kubectl create -f "${SCRIPT_ROOT}/build/crds/devices/devices_v1alpha2_device.yaml"
  fi

  logStatus  "Running integration test cases"
  export KUBE_RACE="-race"
  make -C "${SCRIPT_ROOT}/cloud" test \
      WHAT="${WHAT:-$(findIntegrationTestDirs | paste -sd' ' -)}" \
      GOFLAGS="${GOFLAGS:-}" \
      KUBE_TEST_ARGS="--alsologtostderr=true ${KUBE_TEST_ARGS:-} " \
      KUBE_RACE="$KUBE_RACE" \

  cleanup
}

findIntegrationTestDirs() {
  (
    cd "${SCRIPT_ROOT}/cloud"
    find test/integration/ -name '*_test.go' -print0 \
      | xargs -0n1 dirname | LC_ALL=C sort -u
  )
}

cleanup() {
  logStatus "Cleaning up"

  kubectl get device -n${TESTNS} | tail -n+2 | awk -F' ' '{print $1}' | xargs kubectl delete device -n${TESTNS}
  kubectl get devicemodel -n${TESTNS} | tail -n+2 | awk -F' ' '{print $1}' | xargs kubectl delete devicemodel -n${TESTNS}
  kubectl get crd | tail -n+2 | awk -F' ' '{print $1}' | xargs kubectl delete crd

  kubectl delete namespace ${TESTNS}

  logStatus "Cleanup complete"
}

logStatus() {
  timestamp=$(date +"[%m%d %H:%M:%S]")
  echo "+++ ${timestamp} ${1}"
}

runTests
