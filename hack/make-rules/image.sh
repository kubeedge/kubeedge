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

KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

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
  conformance:conformance:build/conformance/Dockerfile
  controllermanager:controller-manager:build/controllermanager/Dockerfile
  installation-package:installation-package:build/docker/installation-package/installation-package.dockerfile
)

GO_LDFLAGS="$(${KUBEEDGE_ROOT}/hack/make-rules/version.sh)"
IMAGE_TAG=$(git describe --tags)


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

function build_images() {
  local -a targets=()

  for arg in "$@"; do
    targets+=("${arg}")
  done

  if [[ ${#targets[@]} -eq 0 ]]; then
     for bt in "${ALL_IMAGES_AND_TARGETS[@]}" ; do
       targets+=("${bt%%:*}")
     done
  fi

  for arg in "${targets[@]}"; do
    IMAGE_NAME="$(get_imagename_by_target ${arg})"
    DOCKERFILE_PATH="$(get_dockerfile_by_target ${arg})"

    set -x
    docker build --build-arg GO_LDFLAGS="${GO_LDFLAGS}" -t kubeedge/${IMAGE_NAME}:${IMAGE_TAG} -f ${DOCKERFILE_PATH} .
    set +x
  done
}

build_images "$@"
