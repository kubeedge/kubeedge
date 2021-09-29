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

# The root of the build/dist directory
KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
CRD_OUTPUTS="${KUBEEDGE_ROOT}/build/crds"
REQUIRED_VERSION="v1"

echo "Start to check CRDs version..."

for entry in `ls ${CRD_OUTPUTS}/*/*.yaml`; do
	CRD_NAME=`echo ${entry} | awk -F '/' '{print $NF}'`
	if [ "$CRD_NAME" != "devices_v1alpha1_device.yaml" ] && [ "$CRD_NAME" != "devices_v1alpha1_devicemodel.yaml" ]; then
		echo "checking CRD ${CRD_NAME}..."
		version=`cat $entry | grep "apiVersion" | head -1 | awk -F '/' '{print $2}'`
		if [ "$version" != "${REQUIRED_VERSION}" ]; then
			echo "CRD ${CRD_NAME} version does not equal v1"
			exit 1
		fi
	fi
done

echo "Finished, all CRDs version sucessfully..."

