#!/usr/bin/env bash

###
#Copyright 2019 The KubeEdge Authors.
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

# The root of the build/dist directory
KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

function kubeedge::git::check_status() {
	# check if there's any uncommitted changes on go.mod, go.sum or vendor/
	echo $( git status --short 2>/dev/null | grep -E "go.mod|go.sum|vendor/" |wc -l)
}

${KUBEEDGE_ROOT}/hack/update-vendor.sh
 
ret=$(kubeedge::git::check_status)
if [ ${ret} -eq 0 ]; then
	echo "SUCCESS: Vendor Verified."
else
	echo  "FAILED: Vendor Verify failed. Please run the command to check your directories: git status"
	exit 1
fi
