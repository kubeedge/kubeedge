#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
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

TOOL_VERSION="v0.3.4"

# The csi-release-tools directory (absolute path).
TOOLS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

# Directory to check. Default is the parent of the tools themselves.
ROOT="${1:-${TOOLS}/..}"

# create a temporary directory
TMP_DIR=$(mktemp -d)

# cleanup
exitHandler() (
  echo "Cleaning up..."
  rm -rf "${TMP_DIR}"
)
trap exitHandler EXIT

if [[ -z "$(command -v misspell)" ]]; then
  echo "Cannot find misspell. Installing misspell..."
  # perform go get in a temp dir as we are not tracking this version in a go module
  # if we do the go get in the repo, it will create / update a go.mod and go.sum
  cd "${TMP_DIR}"
  GO111MODULE=on GOBIN="${TMP_DIR}" go install "github.com/client9/misspell/cmd/misspell@${TOOL_VERSION}"
  export PATH="${TMP_DIR}:${PATH}"
fi

# check spelling
RES=0
echo "Checking spelling..."
ERROR_LOG="${TMP_DIR}/errors.log"
cd "${ROOT}"
git ls-files | grep -v vendor | xargs misspell > "${ERROR_LOG}"
if [[ -s "${ERROR_LOG}" ]]; then
  sed 's/^/error: /' "${ERROR_LOG}" # add 'error' to each line to highlight in e2e status
  echo "Found spelling errors!"
  RES=1
fi
exit "${RES}"
