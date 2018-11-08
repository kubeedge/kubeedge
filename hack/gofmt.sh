#!/usr/bin/env bash

# Copyright 2018 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# Usage:
#   (1). check and update code format error
#      hack/gofmt.sh -u|--update
#   (2). check the code format error only
#      hack/gofmt.sh -v|--verify

if [ $# -ne 1 ]; then
   echo "usage: $0 <-u|--update|-v|--verify>"
   exit 2
fi

update=false
verify=false
case $1 in
   -u|--update)
      update=true;;
   -v|--verify)
      verify=true;;
   *) echo "Parameter $1 error! Must be one of <-u|--update|-v|--verify>" ; exit 1 ;;
esac

ROOTDIR=$( cd "$(dirname "$0")" ; pwd -P )/..

cd "$ROOTDIR"

GOPATH=$(cd "$ROOTDIR/../.."; pwd)
export GOPATH
export PATH=$GOPATH/bin:$PATH

# The files or packagess we should update/verify gofmt.
find_files() {
  find . -not \( \
      \( \
        -wholename './.git' \
        -o -wholename '*/vendor/*' \
      \) -prune \
    \) -name '*.go'
}

# If run hack/gofmt.sh -u|--update command, will update the format error.
if [[ "${update}" == "true" ]]; then
  GOFMT="gofmt -s -w"
  find_files | xargs $GOFMT
elif [[ "${verify}" == "true" ]]; then
  # If run hack/gofmt.sh -v|--verify command, just throw format error.
  # Hint use hack/gofmt.sh -u|--update command to update format error.
  diff=$(find_files | xargs gofmt -d -s 2>&1) || true
  if [[ -n "${diff}" ]]; then
    echo "${diff}" >&2
    echo >&2
    echo "Run ./hack/gofmt.sh -u|--update" >&2
    exit 1
  fi
fi
