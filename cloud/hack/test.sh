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

KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

kubeedge::golang::get_cloud_test_dirs() {
  (
    local findDirs
    local -a dirArray
    cd ${KUBEEDGE_ROOT}
    findDirs=$(find -L ./cloud -not \( \
        \( \
          -path './cloud/test/integration/*' \
        \) -prune \
      \) -name '*_test.go' -print0 | xargs -0n1 dirname | LC_ALL=C sort -u)
    dirArray=(${findDirs// /})
    echo "${dirArray[@]}"
  )
}

kubeedge::golang::get_edge_test_dirs() {
  (
    local findDirs
    local -a dirArray=()
    cd ${KUBEEDGE_ROOT}
	findDirs=$(find "./edge/pkg" -name "*_test.go"| xargs -I{} dirname {} | uniq)
    dirArray=(${findDirs// /})
    echo "${dirArray[@]}"
  )
}

read -ra KUBEEDGE_CLOUD_TESTCASES <<< "$(kubeedge::golang::get_cloud_test_dirs)"
read -ra KUBEEDGE_EDGE_TESTCASES <<< "$(kubeedge::golang::get_edge_test_dirs)"

readonly KUBEEDGE_ALL_TESTCASES=(
  ${KUBEEDGE_CLOUD_TESTCASES[@]}
  ${KUBEEDGE_EDGE_TESTCASES[@]}
)

ALL_COMPONENTS_AND_GETTESTDIRS_FUNCTIONS=(
  cloud::::kubeedge::golang::get_cloud_test_dirs
  edge::::kubeedge::golang::get_edge_test_dirs
)

kubeedge::golang::get_testdirs_by_component() {
  local key=$1
  for ct in "${ALL_COMPONENTS_AND_GETTESTDIRS_FUNCTIONS[@]}" ; do
    local component="${ct%%::::*}"
    if [ "${component}" == "${key}" ]; then
      local testcases="${ct##*::::}"
      echo $(eval $testcases)
      return
    fi
  done
  echo "can not find component: $key"
  exit 1
}



runTests() {
  echo "running tests cases $@"

  cd ${KUBEEDGE_ROOT}

  local -a testdirs=()
  local binArg
  for binArg in "$@"; do
    testdirs+=("$(kubeedge::golang::get_testdirs_by_component $binArg)")
  done

  if [[ ${#testdirs[@]} -eq 0 ]]; then
    testdirs+=("${KUBEEDGE_ALL_TESTCASES[@]}")
  fi

  go test ${testdirs[@]}

}
runTests $@

