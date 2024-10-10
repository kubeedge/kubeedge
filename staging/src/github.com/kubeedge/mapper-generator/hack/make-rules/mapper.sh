#!/usr/bin/env bash

###
#Copyright 2021 The KubeEdge Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
###

set -o errexit
set -o nounset
set -o pipefail

CURR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
ROOT_DIR="${CURR_DIR}"
source "${ROOT_DIR}/hack/lib/init.sh"

function entry() {
  local mapper="${1}"
  if [[ -f "${ROOT_DIR}/mappers/${mapper}/Makefile" ]]; then
    make -se -f "${ROOT_DIR}/mappers/${mapper}/Makefile" mapper "$@"
  else
    echo "could not find '${mapper}' mapper !!!"
  fi
}

entry "$@"

