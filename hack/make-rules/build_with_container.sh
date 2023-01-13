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

KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
MOUNTPATH="${MOUNTPATH:-/kubeedge}"
KUBEEDGE_BUILD_IMAGE=${KUBEEDGE_BUILD_IMAGE:-"kubeedge/build-tools:1.17.13-ke1"}
DOCKER_GID="${DOCKER_GID:-$(grep '^docker:' /etc/group | cut -f3 -d:)}"
CONTAINER_RUN_OPTIONS="${CONTAINER_RUN_OPTIONS:--it}"

echo "start building inside container"
docker run --rm ${CONTAINER_RUN_OPTIONS} -u "${UID}:${DOCKER_GID}" \
    --init \
    --sig-proxy=true \
    -e XDG_CACHE_HOME=/tmp/.cache \
    -v ${KUBEEDGE_ROOT}:${MOUNTPATH} \
    -w ${MOUNTPATH} ${KUBEEDGE_BUILD_IMAGE} "$@"
