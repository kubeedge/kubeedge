#!/usr/bin/env bash

# Copyright 2019 The KubeEdge Authors.
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

set -e

# set USE_HTTPS to false if you are using git@github.com/xxx/xx.git as remote
USE_HTTPS=${USE_HTTPS:-true}

# set CHECK_GIT_REMOTE false if you already set remote correctly
CHECK_GIT_REMOTE=${CHECK_GIT_REMOTE:-true}

# Useful Defaults
## Target github organization name
TARGET_ORG=${TARGET_ORG:-"kubeedge"}
## main repo uptream configs
UPSTREAM=${UPSTREAM:-"upstream"}
UPSTREAM_HEAD=${UPSTREAM_HEAD:-"master"}
UPSTREAM_REPO_NAME=${UPSTREAM_REPO_NAME:-"kubeedge"}
## out of tree upstream configs
OOT_UPSTREAM=${OOT_UPSTREAM:-"beehive-upstream"}
OOT_UPSTREAM_HEAD=${OOT_UPSTREAM_HEAD:-"master"}
OOT_UPSTREAM_REPO_NAME=${OOT_UPSTREAM_REPO_NAME:-"beehive"}
# sub-directory to sync from upstream project
PATH_TO_SYNC=${PATH_TO_SYNC:-"staging/src/github.com/kubeedge/beehive"}
# branch name for working changes
WORKING_BRANCH_NAME=${WORKING_BRANCH_NAME:-"sync-beehive-code"}


PROTO_HEAD="git@"
SPLITER=":"
if [[ ${USE_HTTPS} ]]; then
  PROTO_HEAD="https://"
  SPLITER="/"
fi

function check-and-add-upstream() {
  local upstream=$1
  local upstream_url=$2
  # git remote -v | grep -qE "^${upstream}\b.*${upstream_url}" || rc="$?"
  git remote get-url ${upstream} | grep -qE "${upstream_url}" || rc="$?"
  if [[ "${rc}" -ne "0" ]]; then
    printf "[INFO] git remote not found, adding: %s\t%s\n" ${upstream} ${upstream_url}
    git remote add ${upstream} ${upstream_url}
  fi
}

if [[ ${CHECK_GIT_REMOTE} ]]; then
  UPSTREAM_URL="${PROTO_HEAD}github.com${SPLITER}${TARGET_ORG}/${UPSTREAM_REPO_NAME}.git"
  OOT_UPSTREAM_URL="${PROTO_HEAD}github.com${SPLITER}${TARGET_ORG}/${OOT_UPSTREAM_REPO_NAME}.git"
  check-and-add-upstream ${UPSTREAM} ${UPSTREAM_URL}
  check-and-add-upstream ${OOT_UPSTREAM} ${OOT_UPSTREAM_URL}
fi

printf "[INFO] updating remote %s, %s.\n" ${UPSTREAM} ${OOT_UPSTREAM}
git remote update --prune ${UPSTREAM} ${OOT_UPSTREAM}

printf "[INFO] creating branch %s based on %s.\n" ${WORKING_BRANCH_NAME} "${UPSTREAM}/${UPSTREAM_HEAD}"
git checkout -b ${WORKING_BRANCH_NAME} --track "${UPSTREAM}/${UPSTREAM_HEAD}"


function sync-code(){
  local PTS=$1
  local target_head=$(git log -1 --pretty=format:"%H" ${OOT_UPSTREAM}/${OOT_UPSTREAM_HEAD})

  # printf "[INFO] checking in %s code with history from %s.\n" ${PTS} ${OOT_UPSTREAM}
  if [ -d ${PTS} ]; then
    #### need test
    local base=$(git log -1 --pretty=format:"%H" ${PTS})
    printf "[INFO] %-20s already exists, updating code to %s.\n" ${PTS} "${OOT_UPSTREAM}/${OOT_UPSTREAM_HEAD}"
    git read-tree --prefix=${PTS} -u ${base}:${PTS} ${target_head}
  else
    printf "[INFO] %-20s not exist, checking in code with history from %s.\n" ${PTS} ${OOT_UPSTREAM}
    mkdir -p ${PTS}
    git read-tree --prefix=${PTS} -u ${target_head}
    git checkout -- ${PTS}
  fi
  printf "[INFO] folders (%s) are now up to date with %s.\n" "${PATHS[*]}" "${target_head}"
}

# Check in code with history from upstream repo
sync-code "${PATH_TO_SYNC}"


NEW_COMMIT=$(git write-tree)
PARENT_A=$(git rev-parse ${WORKING_BRANCH_NAME})
PARENT_B=$(git rev-parse ${OOT_UPSTREAM}/${OOT_UPSTREAM_HEAD})

printf "[INFO] generating merge commit %s with parents:\n" ${NEW_COMMIT}
printf "\t%-20s\t%s\n" ${UPSTREAM} ${PARENT_A}
printf "\t%-20s\t%s\n" ${OOT_UPSTREAM} ${PARENT_B}


# commit subtree updates
FINAL_COMMIT=$(echo "update in-tree ${OOT_UPSTREAM} code" |\
 git commit-tree ${NEW_COMMIT} -p ${PARENT_A} -p ${PARENT_B})

git reset ${FINAL_COMMIT}

printf "[INFO] done.\n"
