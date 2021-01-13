#!/usr/bin/env bash

###
#Copyright 2020 The KubeEdge Authors.
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

KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${KUBEEDGE_ROOT}/hack/lib/init.sh"

kubeedge::clean::cache(){
  unset GOARM
  go clean -testcache
  go clean -cache
}

kubeedge::clean::bin(){
  rm -rf $KUBEEDGE_OUTPUT_BINPATH/*
}

kubeedge::clean::cache
kubeedge::clean::bin
