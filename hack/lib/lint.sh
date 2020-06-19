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

kubeedge::lint::check() {
    cd ${KUBEEDGE_ROOT}
    # skip deleted files
    set +o pipefail
    git diff --cached --name-only --diff-filter=ACRMTU master | grep -Ev "externalversions|fake|vendor|images" | xargs --no-run-if-empty sed -i 's/[ \t]*$//'

    [[ $(git diff --name-only) ]] && {
      echo "Some files have white noise issue, please run `make lint` to check"
      return 1
    }
    golangci-lint run
    gofmt -l -w staging
    set -o pipefail
}
