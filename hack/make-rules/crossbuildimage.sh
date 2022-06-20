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
IMAGE_TAG=$(git describe --tags)
GO_LDFLAGS="$(${KUBEEDGE_ROOT}/hack/make-rules/version.sh)"
IMAGE_REPO_NAME="${IMAGE_REPO_NAME:-kubeedge}"

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

function build_multi_arch_images() {
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

    # If there's any issues when using buildx, can refer to the issue below
    # https://github.com/docker/buildx/issues/495
    # https://github.com/multiarch/qemu-user-static/issues/100
    # docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
    docker buildx build --build-arg GO_LDFLAGS="${GO_LDFLAGS}" -t ${IMAGE_REPO_NAME}/${IMAGE_NAME}:${IMAGE_TAG} -f ${DOCKERFILE_PATH} --platform linux/amd64,linux/arm64,linux/arm/v7 --push .
    set +x
  done
}

#use Docker Buildx to build multi-arch docker images
#How to enable Docker Buildx:
#please follow this to open Buildx function: https://medium.com/@artur.klauser/building-multi-architecture-docker-images-with-buildx-27d80f7e2408
# buildx will push the image to registry, so we need to login registry first and use `-t` flag to set image tag specifically.
build_multi_arch_images "$@"
