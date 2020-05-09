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

kubeedge::lint::cloud_lint() {
  (
    echo "lint cloud"
    cd ${KUBEEDGE_ROOT}/cloud
    golangci-lint run
    go vet ./... 
  )
}

kubeedge::lint::edge_lint() {
  (
    echo "lint edge"
    cd ${KUBEEDGE_ROOT}/edge
    golangci-lint run
    go vet ./...
  )
}

kubeedge::lint::keadm_lint() {
  (
    echo "lint keadm"
    cd ${KUBEEDGE_ROOT}/keadm
    golangci-lint run
    go vet ./...
  )
}

kubeedge::lint::bluetoothdevice_lint() {
  (
    echo "lint bluetoothdevice"
    cd ${KUBEEDGE_ROOT}/mappers/bluetooth_mapper
    golangci-lint run
    go vet ./...
  )
}

kubeedge::lint::global_lint() {
  (
    echo "checking gofmt repo-wide"
    cd ${KUBEEDGE_ROOT}
    golangci-lint run
    go vet ./...
  )
}

ALL_COMPONENTS_AND_LINT_FUNCTIONS=(
  repo::::kubeedge::lint::global_lint
  cloud::::kubeedge::lint::cloud_lint
  edge::::kubeedge::lint::edge_lint
  keadm::::kubeedge::lint::keadm_lint
  bluetoothdevice::::kubeedge::lint::bluetoothdevice_lint
)

kubeedge::lint::get_lintfuntion_by_component() {
  local key=$1
  for cl in "${ALL_COMPONENTS_AND_LINT_FUNCTIONS[@]}" ; do
    local component="${cl%%::::*}"
    if [ "${component}" == "${key}" ]; then
      local func="${cl##*::::}"
      echo "${func}"
      return
    fi
  done
  echo "can not find component: $key"
  exit 1
}

kubeedge::lint::get_all_lintfuntion() {
  local -a funcs 
  for cl in "${ALL_COMPONENTS_AND_LINT_FUNCTIONS[@]}" ; do
    funcs+=("${cl##*::::}")
  done
  echo ${funcs[@]}
}

IFS=" " read -ra ALL_LINT_FUNCTIONS <<< "$(kubeedge::lint::get_all_lintfuntion)"

kubeedge::lint::check() {
  echo "checking golang lint $@"

  cd ${KUBEEDGE_ROOT}

  local -a funcs=()
  local arg
  for arg in "$@"; do
    funcs+=("$(kubeedge::lint::get_lintfuntion_by_component $arg)")
  done

  if [[ ${#funcs[@]} -eq 0 ]]; then
    funcs+=("${ALL_LINT_FUNCTIONS[@]}")
  fi

  for f in ${funcs[@]}; do
    $f
  done
}
