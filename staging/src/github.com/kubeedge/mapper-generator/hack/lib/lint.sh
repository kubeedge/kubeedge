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

SED_CMD=""

if [[ "$OSTYPE" == "darwin"* ]]
then
    SED_CMD=`which gsed`
    if [ -z $SED_CMD ]
    then
        echo "Please install gnu-sed (brew install gnu-sed)"
        exit 1
    fi
elif [[ "$OSTYPE" == "linux"* ]]
then
    SED_CMD=`which sed`
    if [ -z $SED_CMD ]
    then
        echo "Please install sed"
        exit 1
    fi
else
    echo "Unsupported OS $OSTYPE"
    exit 1
fi

kubeedge::lint::check() {
    cd ${KUBEEDGE_ROOT}
    echo "start lint ..."
    set +o pipefail
    echo "check any whitenoise ..."
    # skip deleted files
    if [[ "$OSTYPE" == "darwin"* ]]
    then
      git diff --cached --name-only --diff-filter=ACRMTU main | grep -Ev "externalversions|fake|vendor|images|adopters" | xargs $SED_CMD -i 's/[ \t]*$//'
    elif [[ "$OSTYPE" == "linux"* ]]
    then
      git diff --cached --name-only --diff-filter=ACRMTU main | grep -Ev "externalversions|fake|vendor|images|adopters" | xargs --no-run-if-empty $SED_CMD -i 's/[ \t]*$//'
    else
      echo "Unsupported OS $OSTYPE"
      exit 1
    fi

    [[ $(git diff --name-only) ]] && {
      echo "Some files have white noise issue, please run \`make lint\` to slove this issue"
      return 1
    }
    set -o pipefail

    echo "check any issue by golangci-lint ..."
    GOOS="linux" golangci-lint run -v
}
