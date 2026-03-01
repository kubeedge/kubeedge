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
    local -a files_to_check=()
    local file
    while IFS= read -r file; do
      [[ -n "${file}" ]] && files_to_check+=("${file}")
    done < <(git diff --cached --name-only --diff-filter=ACRMTU master | grep -Ev "externalversions|fake|vendor|images|adopters" || true)

    local before_hashes
    local after_hashes
    before_hashes=$(mktemp)
    after_hashes=$(mktemp)
    trap 'rm -f "${before_hashes}" "${after_hashes}"' RETURN

    for file in "${files_to_check[@]}"; do
      if [[ -f "${file}" ]]; then
        printf '%s\t%s\n' "$(git hash-object "${file}")" "${file}" >> "${before_hashes}"
      fi
    done

    if [[ "$OSTYPE" == "darwin"* ]]
    then
      if [[ ${#files_to_check[@]} -gt 0 ]]; then
        printf '%s\n' "${files_to_check[@]}" | xargs $SED_CMD -i 's/[ \t]*$//'
      fi
    elif [[ "$OSTYPE" == "linux"* ]]
    then
      printf '%s\n' "${files_to_check[@]}" | xargs --no-run-if-empty $SED_CMD -i 's/[ \t]*$//'
    else
      echo "Unsupported OS $OSTYPE"
      exit 1
    fi

    for file in "${files_to_check[@]}"; do
      if [[ -f "${file}" ]]; then
        printf '%s\t%s\n' "$(git hash-object "${file}")" "${file}" >> "${after_hashes}"
      fi
    done

    local -a white_noise_files=()
    local before_hash
    local after_hash
    while IFS=$'\t' read -r before_hash file; do
      after_hash=$(awk -F '\t' -v f="${file}" '$2 == f {print $1; exit}' "${after_hashes}")
      if [[ "${before_hash}" != "${after_hash}" ]]; then
        white_noise_files+=("${file}")
      fi
    done < "${before_hashes}"

    [[ ${#white_noise_files[@]} -gt 0 ]] && {
      echo "Some files have white noise issue, please run \`make lint\` to solve and \`git status\` to find and fix this issue"
      printf 'white noise fixed in files:\n'
      printf '  %s\n' "${white_noise_files[@]}"
      return 1
    }
    set -o pipefail

    echo "check any issue by golangci-lint ..."
    GOOS="linux" golangci-lint run -v --timeout=12m

    # check codes under staging dir, this will also use .golangci.yaml in the {KUBEEDGE_ROOT} dir
    echo "check any issue by golangci-lint under staging dir ..."
    cd "${KUBEEDGE_ROOT}/staging/src/github.com/kubeedge/beehive" && GOOS="linux" golangci-lint run -v --timeout=1m
    cd "${KUBEEDGE_ROOT}/staging/src/github.com/kubeedge/mapper-framework" && GOOS="linux" golangci-lint run -v --timeout=2m
}
