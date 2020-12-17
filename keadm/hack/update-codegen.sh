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

KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
TMP_DIR="${KUBEEDGE_ROOT}/_tmp"

LATEST_VERSION=$(curl https://kubeedge.io/latestversion 2>/dev/null | cut -b 2-4)
SUPPORT_VERSION="$(echo ${LATEST_VERSION} - 0.2 | bc)
$(echo ${LATEST_VERSION} - 0.1 | bc)
${LATEST_VERSION}"

cd $KUBEEDGE_ROOT

cleanup() {
  rm -rf $TMP_DIR
}

trap cleanup EXIT SIGINT

cleanup
mkdir -p ${TMP_DIR}

origin_place=$(git branch | grep '\*')
if echo $origin_place | grep -q '('; then
  origin_place=$(git rev-parse HEAD)
fi

# set upstream
upstream_repo="https://github.com/kubeedge/kubeedge.git"
if git remote -v | grep -q upstream; then
  git remote set-url upstream ${upstream_repo}
else
  git remote add upstream ${upstream_repo}
fi

for version in ${SUPPORT_VERSION}; do
  crds_dir=${version}/crds
  services_dir=${version}/services

  mkdir -p $crds_dir $services_dir

  # get files in each version
  git checkout "release-${version}"

  # 1.3 still use v1alpha1
  if [[ ${version} == "1.3" ]]; then
    cp build/crds/devices/*v1alpha1* build/crds/reliablesyncs/* $crds_dir
  else
    cp build/crds/devices/*v1alpha2* build/crds/reliablesyncs/* $crds_dir
  fi

  cp build/tools/*.service $services_dir
done

# rollback
git checkout $origin_place

command -v go-bindata &>/dev/null || go get -u github.com/go-bindata/go-bindata/...
# revert module changes
git checkout .

mv $(echo $SUPPORT_VERSION) ${TMP_DIR}

embed_file="../keadm/cmd/keadm/app/cmd/common/embed.go"
cd ${TMP_DIR}
go-bindata -nometadata -pkg common -o $embed_file ./...
