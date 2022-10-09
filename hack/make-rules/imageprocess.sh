#!/usr/bin/env bash

###
#Copyright 2022 The KubeEdge Authors.
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

Parameter_imagename="imagename"
Parameter_dockerfile="dockerfile"

ALL_IMAGES_AND_TARGETS=(
  #{target}:{IMAGE_NAME}:{DOCKERFILE_PATH}
  cloudcore:cloudcore:build/cloud/Dockerfile
  admission:admission:build/admission/Dockerfile
  edgecore:edgecore:build/edge/Dockerfile
  edgesite-agent:edgesite-agent:build/edgesite/agent-build.Dockerfile
  edgesite-server:edgesite-server:build/edgesite/server-build.Dockerfile
  csidriver:csidriver:build/csidriver/Dockerfile
  iptablesmanager:iptables-manager:build/iptablesmanager/Dockerfile
  edgemark:edgemark:build/edgemark/Dockerfile
  installation-package:installation-package:build/docker/installation-package/installation-package.dockerfile
  controllermanager:controller-manager:build/controllermanager/Dockerfile
)

function get_imagename_by_target() {
  local key=$1
  for bt in "${ALL_IMAGES_AND_TARGETS[@]}" ; do
    local binary="${bt%%:*}"
    if [ "${binary}" == "${key}" ]; then
      local name_path="${bt#*:}"
      echo "${name_path%%:*}"
      return
    fi
  done
  echo "can not find image name: $key"
  exit 1
}

function get_dockerfile_by_target() {
  local key=$1
  for bt in "${ALL_IMAGES_AND_TARGETS[@]}" ; do
    local binary="${bt%%:*}"
    if [ "${binary}" == "${key}" ]; then
      local name_path="${bt#*:}"
      echo "${name_path#*:}"
      return
    fi
  done
  echo "can not find dockerfile for: $key"
  exit 1
}

if [[ $1 == $Parameter_imagename ]]; then
   echo $(get_imagename_by_target $2)
elif [[ $1 == $Parameter_dockerfile ]]; then
    echo $(get_dockerfile_by_target $2)
fi
