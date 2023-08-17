#!/usr/bin/env bash

# Copyright 2020 The KubeEdge Authors.
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

# The root of the kubeedge
KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

KUBEEDGE_OUTPUT_SUBPATH="${KUBEEDGE_OUTPUT_SUBPATH:-_output/local}"
KUBEEDGE_OUTPUT="${KUBEEDGE_ROOT}/${KUBEEDGE_OUTPUT_SUBPATH}"
KUBEEDGE_OUTPUT_BINPATH="${KUBEEDGE_OUTPUT}/bin"

readonly KUBEEDGE_GO_PACKAGE="github.com/kubeedge/mappers-go"

source "${KUBEEDGE_ROOT}/hack/lib/lint.sh"
source "${KUBEEDGE_ROOT}/hack/lib/util.sh"
