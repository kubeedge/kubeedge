#!/usr/bin/env bash

# Copyright 2024 The KubeEdge Authors.
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

KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
#!/usr/bin/env bash

# Initialize _tmp variable
_tmp=$(mktemp -d)


cleanup() {
  chmod -x "${KUBEEDGE_ROOT}/hack/verify-apidoc.sh"

  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

cleanup
chmod +x "${KUBEEDGE_ROOT}/hack/verify-apidoc.sh"

DIFFROOT="${KUBEEDGE_ROOT}/staging/src/github.com/kubeedge/api"
TMP_DIFFROOT="${KUBEEDGE_ROOT}/_tmp/api"
mkdir -p "${TMP_DIFFROOT}"
cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

sudo bash "${KUBEEDGE_ROOT}/staging/api/github.com/kubeedge/api/apidoc/tools/generate-openapi.sh"
sudo bash "${KUBEEDGE_ROOT}/staging/src/github.com/kubeedge/api/apidoc/tools/update-swagger-docs.sh"
echo "diffing ${DIFFROOT} against freshly generated swagger docs"
ret=0
diff -Naupr "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?
cp -a "${TMP_DIFFROOT}"/* "${DIFFROOT}"
if [[ $ret -eq 0 ]]
then
  echo "${DIFFROOT} up to date."
else
  echo "${DIFFROOT} is out of date. Please update swagger docs "
  exit 1
fi
