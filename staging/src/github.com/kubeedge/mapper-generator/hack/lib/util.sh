#!/usr/bin/env bash

# Copyright 2014 The Kubernetes Authors.
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
#
# -----------------------------------------------------------------------------
# CHANGELOG
# KubeEdge Authors:
# Some functions derived from https://github.com/kubernetes/kubernetes/blob/v1.19.0-beta.2/hack/utils.sh
# for update-vendor-licenses, verify-vendor-licenses


# Example:  kube::util::trap_add 'echo "in trap DEBUG"' DEBUG
# See: http://stackoverflow.com/questions/3338030/multiple-bash-traps-for-the-same-signal
kube::util::trap_add() {
  local trap_add_cmd
  trap_add_cmd=$1
  shift

  for trap_add_name in "$@"; do
    local existing_cmd
    local new_cmd

    # Grab the currently defined trap commands for this trap
    existing_cmd=$(trap -p "${trap_add_name}" |  awk -F"'" '{print $2}')

    if [[ -z "${existing_cmd}" ]]; then
      new_cmd="${trap_add_cmd}"
    else
      new_cmd="${trap_add_cmd};${existing_cmd}"
    fi

    # Assign the test. Disable the shellcheck warning telling that trap
    # commands should be single quoted to avoid evaluating them at this
    # point instead evaluating them at run time. The logic of adding new
    # commands to a single trap requires them to be evaluated right away.
    # shellcheck disable=SC2064
    trap "${new_cmd}" "${trap_add_name}"
  done
}

# Opposite of kube::util::ensure-temp-dir()
kube::util::cleanup-temp-dir() {
  rm -rf "${KUBE_TEMP}"
}

# Create a temp dir that'll be deleted at the end of this bash session.
#
# Vars set:
#   KUBE_TEMP
kube::util::ensure-temp-dir() {
  if [[ -z ${KUBE_TEMP-} ]]; then
    KUBE_TEMP=$(mktemp -d 2>/dev/null || mktemp -d -t kubernetes.XXXXXX)
    kube::util::trap_add kube::util::cleanup-temp-dir EXIT
  fi
}

# outputs md5 hash of $1, works on macOS and Linux
function kube::util::md5() {
  if which md5 >/dev/null 2>&1; then
    md5 -q "$1"
  else
    md5sum "$1" | awk '{ print $1 }'
  fi
}
