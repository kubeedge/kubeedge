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

KUBE_RACE=${KUBE_RACE:-}
eval "testargs=(${KUBE_TEST_ARGS:-})"
SCRIPT_ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/../../ && pwd)

findTestDirs() {
  (
    cd ${SCRIPT_ROOT}/cloud
    find -L . -not \( \
        \( \
          -path './test/integration/*' \
        \) -prune \
      \) -name '*_test.go' -print0 | xargs -0n1 dirname | LC_ALL=C sort -u
  )
}

testcases=()
for arg; do
  if [[ "${arg}" == -* ]]; then
    goflags+=("${arg}")
  else
    testcases+=("${arg}")
  fi
done
if [[ ${#testcases[@]} -eq 0 ]]; then
  testcases=($(findTestDirs))
fi
set -- "${testcases[@]+${testcases[@]}}"

runTests() {
  cd ${SCRIPT_ROOT}/cloud
  go test "${goflags[@]:+${goflags[@]}}" ${KUBE_RACE} "./${@}" \
  "${testargs[@]:+${testargs[@]}}"
}

runTests "$@"
