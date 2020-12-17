#!/bin/bash

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

SCRIPT_ROOT="$(dirname "${BASH_SOURCE}")/.."

TMP_DIR="${SCRIPT_ROOT}/_tmp"

cleanup() {
  rm -rf ${TMP_DIR}
}

trap cleanup EXIT SIGINT

cleanup

${SCRIPT_ROOT}/hack/update-codegen.sh

EMBED_FILE="keadm/cmd/keadm/app/cmd/common/embed.go"

if [[ $(git diff --name-only) ]]; then
  echo "${EMBED_FILE} is out of date. Please run cloud/hack/update-codegen.sh"
  exit 1
else
  echo "${EMBED_FILE} is up to date."
fi
