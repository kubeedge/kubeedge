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

function kubeedge::lint::init() {
    SED_CMD=""
    KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

    if [[ "$OSTYPE" == "darwin"* ]]
    then
        if ! which gsed >/dev/null 2>&1
        then
            echo "Please install gnu-sed (brew install gnu-sed)"
            exit 1
        else
            SED_CMD=`which gsed`
        fi
    elif [[ "$OSTYPE" == "linux"* ]]
    then
        if ! which sed >/dev/null 2>&1
        then
            echo "Please install sed"
            exit 1
        else
            SED_CMD=`which sed`
        fi
    else
        echo "Unsupported OS $OSTYPE"
        exit 1
    fi
}

kubeedge::lint::check() {
    kubeedge::lint::init
    cd ${KUBEEDGE_ROOT}
    echo "start lint ..."
    set +o pipefail
    echo "check any white noise ..."
    # skip deleted files
    if [[ "$OSTYPE" == "darwin"* ]]
    then
      git diff --cached --name-only --diff-filter=ACRMTU master | grep -Ev "externalversions|fake|vendor|images|adopters" | xargs $SED_CMD -i 's/[ \t]*$//'
    elif [[ "$OSTYPE" == "linux"* ]]
    then
      git diff --cached --name-only --diff-filter=ACRMTU master | grep -Ev "externalversions|fake|vendor|images|adopters" | xargs --no-run-if-empty $SED_CMD -i 's/[ \t]*$//'
    else
      echo "Unsupported OS $OSTYPE"
      exit 1
    fi

    [[ $(git diff --name-only) ]] && {
      echo "Some files have white noise issue, please run \`make lint\` to solve and \`git status\` to find and fix this issue"
      return 1
    }
    set -o pipefail

    echo "check any issue by golangci-lint ..."
    GOOS="linux" golangci-lint run -v

    # check codes under staging dir, this will also use .golangci.yaml in the {KUBEEDGE_ROOT} dir
    cd "${KUBEEDGE_ROOT}/staging/src/github.com/kubeedge/beehive" && GOOS="linux" golangci-lint run -v
    cd "${KUBEEDGE_ROOT}/staging/src/github.com/kubeedge/viaduct" && GOOS="linux" golangci-lint run -v
}
