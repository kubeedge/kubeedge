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
GO_LDFLAGS="$(bash ${KUBEEDGE_ROOT}/hack/make-rules/version.sh)"

SUPPORT_ARCHS="amd64
arm
arm64
"

ALL_BINARIES_AND_TARGETS=(
  cloudcore:kubeedge/cloud/cloudcore
  admission:kubeedge/cloud/admission
  csidriver:kubeedge/cloud/csidriver
  keadm:keadm
  edgecore:kubeedge/edge
  edgesite:edgesite/edgesite
)

kubeedge::release::get_target_by_binary() {
  local key=$1
  for bt in "${ALL_BINARIES_AND_TARGETS[@]}" ; do
    local binary="${bt%%:*}"
    if [ "${binary}" == "${key}" ]; then
			echo "${bt##*:}"
			return
    fi
  done
  echo "can not find binary: $key"
	exit 1
}

kubeedge::release::build_all_binary() {
  for arch in ${SUPPORT_ARCHS}; do
    ARCH=$arch kubeedge::golang::crossbuild_binaries
  done

  for dir in ${KUBEEDGE_OUTPUT_BINPATH}/*; do
    local arch=${dir##*/}
    pushd $dir >/dev/null
    for file in *; do
      local target=$(kubeedge::release::get_target_by_binary ${file})
      local file_dir="${target%%/*}-${VERSION}-linux-${arch}/${target#*/}"
      mkdir -p ${file_dir}
      echo ${VERSION} > ${file_dir}/version
      mv ${file} ${file_dir}
    done

    for d in *; do
      tar -czf ${d}.tar.gz ${d}
      sha256sum "${d}.tar.gz" > "checksum_${d}.tar.gz.txt"
    done

    popd >/dev/null
  done

  mkdir output
  cp -a ${KUBEEDGE_OUTPUT_BINPATH}/**/*.tar.gz* output/
}
