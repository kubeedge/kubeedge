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

# -----------------------------------------------------------------------------
# CHANGELOG
# KubeEdge Authors:
# To Get Detail Version Info for KubeEdge Project

set -o errexit
set -o nounset
set -o pipefail

YES="y"
NO="n"
KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

kubeedge::version::get_version_info() {

  GIT_COMMIT=$(git rev-parse "HEAD^{commit}" 2>/dev/null)

  if git_status=$(git status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
    GIT_TREE_STATE="clean"
  else
    GIT_TREE_STATE="dirty"
  fi

  GIT_VERSION=$(git describe --tags --abbrev=14 "${GIT_COMMIT}^{commit}" 2>/dev/null)

  # This translates the "git describe" to an actual semver.org
  # compatible semantic version that looks something like this:
  #   v1.1.0-alpha.0.6+84c76d1142ea4d
  #
  # TODO: We continue calling this "git version" because so many
  # downstream consumers are expecting it there.
  #
  # These regexes are painful enough in sed...
  # We don't want to do them in pure shell, so disable SC2001
  # shellcheck disable=SC2001
  DASHES_IN_VERSION=$(echo "${GIT_VERSION}" | sed "s/[^-]//g")
  if [[ "${DASHES_IN_VERSION}" == "---" ]] ; then
    # shellcheck disable=SC2001
    # We have distance to subversion (v1.1.0-subversion-1-gCommitHash)
    GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-\([0-9]\{1,\}\)-g\([0-9a-f]\{14\}\)$/.\1\+\2/")
  elif [[ "${DASHES_IN_VERSION}" == "--" ]] ; then
      # shellcheck disable=SC2001
      # We have distance to base tag (v1.1.0-1-gCommitHash)
      GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-g\([0-9a-f]\{14\}\)$/+\1/")
  fi

  if [[ "${GIT_TREE_STATE}" == "dirty" ]]; then
    # git describe --dirty only considers changes to existing files, but
    # that is problematic since new untracked .go files affect the build,
    # so use our idea of "dirty" from git status instead.
    GIT_VERSION+="-dirty"
  fi


  # Try to match the "git describe" output to a regex to try to extract
  # the "major" and "minor" versions and whether this is the exact tagged
  # version or whether the tree is between two tagged versions.
  if [[ "${GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?([-].*)?([+].*)?$ ]]; then
    GIT_MAJOR=${BASH_REMATCH[1]}
    GIT_MINOR=${BASH_REMATCH[2]}
    if [[ -n "${BASH_REMATCH[4]}" ]]; then
      GIT_MINOR+="+"
    fi
  fi

  # If GIT_VERSION is not a valid Semantic Version, then refuse to build.
  if ! [[ "${GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
      echo "GIT_VERSION should be a valid Semantic Version. Current value: ${GIT_VERSION}"
      echo "Please see more details here: https://semver.org"
      exit 1
  fi
}

# Get the value that needs to be passed to the -ldflags parameter of go build
kubeedge::version::ldflags() {
  kubeedge::version::get_version_info

  local -a ldflags
  function add_ldflag() {
    local key=${1}
    local val=${2}
    # If you update these, also update the list pkg/version/def.bzl.
    ldflags+=(
      "-X ${KUBEEDGE_GO_PACKAGE}/pkg/version.${key}=${val}"
    )
  }

  add_ldflag "buildDate" "$(date ${SOURCE_DATE_EPOCH:+"--date=@${SOURCE_DATE_EPOCH}"} -u +'%Y-%m-%dT%H:%M:%SZ')"
  if [[ -n ${GIT_COMMIT-} ]]; then
    add_ldflag "gitCommit" "${GIT_COMMIT}"
    add_ldflag "gitTreeState" "${GIT_TREE_STATE}"
  fi

  if [[ -n ${GIT_VERSION-} ]]; then
    add_ldflag "gitVersion" "${GIT_VERSION}"
  fi

  if [[ -n ${GIT_MAJOR-} && -n ${GIT_MINOR-} ]]; then
    add_ldflag "gitMajor" "${GIT_MAJOR}"
    add_ldflag "gitMinor" "${GIT_MINOR}"
  fi

  # The -ldflags parameter takes a single string, so join the output.
  echo "${ldflags[*]-}"
}


# kubeedge::binaries_from_targets take a list of build targets and return the
# full go package to be built
kubeedge::golang::binaries_from_targets() {
  local target
  for target in "$@"; do
    echo "${KUBEEDGE_GO_PACKAGE}/${target}"
  done
}

kubeedge::check::env() {
  errors=()
  if [ -z $GOPATH ]; then
    errors+="GOPATH environment value not set"
  fi

  # check other env

  # check lenth of errors
  if [[ ${#errors[@]} -ne 0 ]] ; then
    local error
    for error in ${errors[@]}; do
      echo "Error: "$error
    done
    exit 1
  fi
}

ALL_BINARIES_AND_TARGETS=(
  cloudcore:cloud/cmd/cloudcore
  admission:cloud/cmd/admission
  csidriver:cloud/cmd/csidriver
  keadm:keadm/cmd/keadm
  edgecore:edge/cmd/edgecore
  edgesite:edgesite/cmd/edgesite
)

kubeedge::golang::get_target_by_binary() {
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

kubeedge::golang::get_all_targets() {
  local -a targets
  for bt in "${ALL_BINARIES_AND_TARGETS[@]}" ; do
    targets+=("${bt##*:}")
  done
  echo ${targets[@]}
}

kubeedge::golang::get_all_binares() {
  local -a binares
  for bt in "${ALL_BINARIES_AND_TARGETS[@]}" ; do
    binares+=("${bt%%:*}")
  done
  echo ${binares[@]}
}

IFS=" " read -ra KUBEEDGE_ALL_TARGETS <<< "$(kubeedge::golang::get_all_targets)"
IFS=" " read -ra KUBEEDGE_ALL_BINARIES<<< "$(kubeedge::golang::get_all_binares)"

kubeedge::golang::build_binaries() {
  kubeedge::check::env
  local -a targets=()
  local binArg
  for binArg in "$@"; do
    targets+=("$(kubeedge::golang::get_target_by_binary $binArg)")
  done

  if [[ ${#targets[@]} -eq 0 ]]; then
    targets=("${KUBEEDGE_ALL_TARGETS[@]}")
  fi

  local -a binaries
  while IFS="" read -r binary; do binaries+=("$binary"); done < <(kubeedge::golang::binaries_from_targets "${targets[@]}")

  local ldflags
  read -r ldflags <<< "$(kubeedge::version::ldflags)"

  local build_option=${BUILD_OPTION:-}

  # do not build small binary in CI env
  local ci_env=${CI_ENV:-No}
  [[ ${ci_env} == "No" ]] && ldflags="-w -s -extldflags -static $ldflags"

  mkdir -p ${KUBEEDGE_OUTPUT_BINPATH} .build

  docker rm -f kubeedge_build &>/dev/null || true

  docker run -itd --name kubeedge_build -v ${KUBEEDGE_ROOT}:/go/src/github.com/kubeedge/kubeedge \
    -w /go/src/github.com/kubeedge/kubeedge golang:1.13-stretch
  docker exec -i kubeedge_build bash -c "go mod download; apt-get update && apt-get install -y gcc-aarch64-linux-gnu gcc-arm-linux-gnueabi"

  for bin in ${binaries[@]}; do
    echo "building $bin"
    local name="${bin##*/}"
    set -x
    # sqlite need cgo indeed
    docker exec -i kubeedge_build bash -c "GOOS=linux CGO_ENABLED=1 ${build_option} \
      go build -o .build/${name} -ldflags '$ldflags' $bin"
    # do not build small binary in CI env
    [[ ${ci_env} == "No" ]] && upx-ucl -9 .build/${name}
  done

  mv .build/* ${KUBEEDGE_OUTPUT_BINPATH}
  docker rm -f kubeedge_build &>/dev/null
  rm -rf .build
}

kubeedge::golang::crossbuild_binaries() {
  case $ARCH in
    arm)
      BUILD_OPTION="CC=arm-linux-gnueabi-gcc GOARCH=arm GOARM=7" kubeedge::golang::build_binaries "$@"
      ;;
    arm64)
      BUILD_OPTION="CC=aarch64-linux-gnu-gcc GOARCH=arm64" kubeedge::golang::build_binaries "$@"
      ;;
    *)
      kubeedge::golang::build_binaries "$@"
      ;;
  esac

  local arch_dir=${KUBEEDGE_OUTPUT_BINPATH}/${ARCH}
  mkdir -p ${KUBEEDGE_OUTPUT_BINPATH}/${ARCH}
  for file in ${KUBEEDGE_OUTPUT_BINPATH}/*; do
    [[ -d $file ]] || mv -- ${file} ${arch_dir}/
  done
}

kubeedge::golang::get_cloud_test_dirs() {
  (
    local findDirs
    local -a dirArray
    cd ${KUBEEDGE_ROOT}
    findDirs=$(find -L ./cloud -not \( \
        \( \
          -path './cloud/test/integration/*' \
        \) -prune \
      \) -name '*_test.go' -print0 | xargs -0n1 dirname | LC_ALL=C sort -u)
    dirArray=(${findDirs// /})
    echo "${dirArray[@]}"
  )
}

kubeedge::golang::get_keadm_test_dirs() {
    cd ${KUBEEDGE_ROOT}
    findDirs=$(find -L ./keadm \
	    -name '*_test.go' -print | xargs -0n1 dirname | uniq)
    echo "${findDirs}"
}

kubeedge::golang::get_edge_test_dirs() {
  (
    local findDirs
    local -a dirArray=()
    cd ${KUBEEDGE_ROOT}
    findDirs=$(find "./edge/pkg" -name "*_test.go"| xargs -I{} dirname {} | uniq)
    dirArray=(${findDirs// /})
    echo "${dirArray[@]}"
  )
}

read -ra KUBEEDGE_CLOUD_TESTCASES <<< "$(kubeedge::golang::get_cloud_test_dirs)"
read -ra KUBEEDGE_EDGE_TESTCASES <<< "$(kubeedge::golang::get_edge_test_dirs)"
read -ra KUBEEDGE_KEADM_TESTCASES <<< "$(kubeedge::golang::get_keadm_test_dirs)"

readonly KUBEEDGE_ALL_TESTCASES=(
  ${KUBEEDGE_CLOUD_TESTCASES[@]}
  ${KUBEEDGE_EDGE_TESTCASES[@]}
  ${KUBEEDGE_KEADM_TESTCASES[@]}
)

ALL_COMPONENTS_AND_GETTESTDIRS_FUNCTIONS=(
  cloud::::kubeedge::golang::get_cloud_test_dirs
  edge::::kubeedge::golang::get_edge_test_dirs
  keadm::::kubeedge::golang::get_keadm_test_dirs
)

kubeedge::golang::get_testdirs_by_component() {
  local key=$1
  for ct in "${ALL_COMPONENTS_AND_GETTESTDIRS_FUNCTIONS[@]}" ; do
    local component="${ct%%::::*}"
    if [ "${component}" == "${key}" ]; then
      local testcases="${ct##*::::}"
      echo $(eval $testcases)
      return
    fi
  done
  echo "can not find component: $key"
  exit 1
}

kubeedge::golang::run_test() {
  echo "running tests cases $@"

  cd ${KUBEEDGE_ROOT}

  local -a testdirs=()
  local binArg
  for binArg in "$@"; do
    testdirs+=("$(kubeedge::golang::get_testdirs_by_component $binArg)")
  done

  if [[ ${#testdirs[@]} -eq 0 ]]; then
    testdirs+=("${KUBEEDGE_ALL_TESTCASES[@]}")
  fi

  go test ${testdirs[@]}
}
