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

export GOPATH="${GOPATH:-$(go env GOPATH)}"

go_path="${KUBEEDGE_ROOT}/_go"
cleanup() {
  rm -rf "${go_path}"
}
trap "cleanup" EXIT SIGINT

cleanup

source "${KUBEEDGE_ROOT}"/hack/lib/util.sh
util:create_gopath_tree "${KUBEEDGE_ROOT}" "${go_path}"
export GOPATH="${go_path}"

${KUBEEDGE_ROOT}/hack/generate-groups.sh "deepcopy,client,informer,lister" \
github.com/kubeedge/kubeedge/pkg/client github.com/kubeedge/kubeedge/pkg/apis \
"devices:v1beta1 reliablesyncs:v1alpha1 rules:v1 apps:v1alpha1 operations:v1alpha1 policy:v1alpha1" \
--go-header-file ${KUBEEDGE_ROOT}/hack/boilerplate/boilerplate.txt
