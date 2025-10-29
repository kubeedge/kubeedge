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

KUBEEDGE_ROOT=$(unset CDPATH && cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)

source "${KUBEEDGE_ROOT}/hack/lib/init.sh"
kubeedge::golang::setup_env

API_VERSIONS=(
    "devices:v1beta1"
    "reliablesyncs:v1alpha1"
    "rules:v1"
    "streamrules:v1alpha1"
    "apps:v1alpha1"
    "operations:v1alpha1"
    "operations:v1alpha2"
    "policy:v1alpha1"
)

function api_versions_tostring() {
    local res=""
    for i in ${API_VERSIONS[@]}; do
        if [[ -n "$res" ]]; then
            res+=" "
        fi
        res+=$i
    done
    echo $res
}

${KUBEEDGE_ROOT}/hack/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/kubeedge/api/client github.com/kubeedge/api/apis \
  "$(api_versions_tostring)" \
  --go-header-file ${KUBEEDGE_ROOT}/hack/boilerplate/boilerplate.txt
