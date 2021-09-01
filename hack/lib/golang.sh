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

  # check length of errors
  if [[ ${#errors[@]} -ne 0 ]] ; then
    local error
    for error in "${errors[@]}"; do
      echo "Error: "$error
    done
    exit 1
  fi
}

ALL_BINARIES_AND_TARGETS=(
  cloudcore:cloud/cmd/cloudcore
  admission:cloud/cmd/admission
  keadm:keadm/cmd/keadm
  edgecore:edge/cmd/edgecore
  edgesite-agent:edgesite/cmd/edgesite-agent
  edgesite-server:edgesite/cmd/edgesite-server
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

kubeedge::golang::get_all_binaries() {
  local -a binaries
  for bt in "${ALL_BINARIES_AND_TARGETS[@]}" ; do
    binaries+=("${bt%%:*}")
  done
  echo ${binaries[@]}
}

IFS=" " read -ra KUBEEDGE_ALL_TARGETS <<< "$(kubeedge::golang::get_all_targets)"
IFS=" " read -ra KUBEEDGE_ALL_BINARIES<<< "$(kubeedge::golang::get_all_binaries)"

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

  local goldflags gogcflags
  # If GOLDFLAGS is unset, then set it to the a default of "-s -w".
  goldflags="${GOLDFLAGS=-s -w -buildid=} $(kubeedge::version::ldflags)"
  gogcflags="${GOGCFLAGS:-}"

  mkdir -p ${KUBEEDGE_OUTPUT_BINPATH}
  for bin in ${binaries[@]}; do
    echo "building $bin"
    local name="${bin##*/}"
    set -x
    go build -o ${KUBEEDGE_OUTPUT_BINPATH}/${name} -gcflags="${gogcflags:-}" -ldflags "${goldflags:-}" $bin
    set +x
  done

}


KUBEEDGE_ALL_CROSS_BINARIES=(
edgecore
)

kubeedge::golang::is_cross_build_binary() {
  local key=$1
  for bin in "${KUBEEDGE_ALL_CROSS_BINARIES[@]}" ; do
    if [ "${bin}" == "${key}" ]; then
      echo ${YES}
      return
    fi
  done
  echo ${NO}
}

KUBEEDGE_ALL_CROSS_GOARMS=(
8
7
)

kubeedge::golang::is_supported_goarm() {
  local key=$1
  for value in ${KUBEEDGE_ALL_CROSS_GOARMS[@]} ; do
    if [ "${value}" == "${key}" ]; then
      echo ${YES}
      return
    fi
  done
  echo ${NO}
}

kubeedge::golang::cross_build_place_binaries() {
  kubeedge::check::env

  local -a targets=()
  local goarm=${goarm:-${KUBEEDGE_ALL_CROSS_GOARMS[0]}}

  for arg in "$@"; do
      if [[ "${arg}" == GOARM* ]]; then
        # Assume arguments starting with a dash are flags to pass to go.
        goarm="${arg##*GOARM}"
      else
        if [ "$(kubeedge::golang::is_cross_build_binary ${arg})" == "${NO}" ]; then
          echo "${arg} does not support cross build"
          exit 1
        fi
        targets+=("$(kubeedge::golang::get_target_by_binary $arg)")
      fi
  done

  if [[ ${#targets[@]} -eq 0 ]]; then
    for bin in ${KUBEEDGE_ALL_CROSS_BINARIES[@]}; do
        targets+=("$(kubeedge::golang::get_target_by_binary $bin)")
    done
  fi

  if [ "$(kubeedge::golang::is_supported_goarm ${goarm})" == "${NO}" ]; then
    echo "GOARM${goarm} does not support cross build"
    exit 1
  fi

  local -a binaries
  while IFS="" read -r binary; do binaries+=("$binary"); done < <(kubeedge::golang::binaries_from_targets "${targets[@]}")

  local ldflags
  read -r ldflags <<< "$(kubeedge::version::ldflags)"

  mkdir -p ${KUBEEDGE_OUTPUT_BINPATH}
  for bin in ${binaries[@]}; do
    echo "cross buildding $bin GOARM${goarm}"
    local name="${bin##*/}"
    if [ "${goarm}" == "8" ]; then
      set -x
      GOARCH=arm64 GOOS="linux" CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build -o ${KUBEEDGE_OUTPUT_BINPATH}/${name} -ldflags "$ldflags" $bin
      set +x
    elif [ "${goarm}" == "7" ]; then
      set -x
      GOARCH=arm GOOS="linux" GOARM=${goarm} CGO_ENABLED=1 CC=arm-linux-gnueabi-gcc go build -o ${KUBEEDGE_OUTPUT_BINPATH}/${name} -ldflags "$ldflags" $bin
      set +x
    fi
  done
}

KUBEEDGE_ALL_SMALL_BINARIES=(
edgecore
)

kubeedge::golang::is_small_build_binary() {
  local key=$1
  for bin in "${KUBEEDGE_ALL_SMALL_BINARIES[@]}" ; do
    if [ "${bin}" == "${key}" ]; then
      echo ${YES}
      return
    fi
  done
  echo ${NO}
}

kubeedge::golang::small_build_place_binaries() {
  kubeedge::check::env
  local -a targets=()

  for arg in "$@"; do
    if [ "$(kubeedge::golang::is_small_build_binary ${arg})" == "${NO}" ]; then
      echo "${arg} does not support small build"
      exit 1
    fi
    targets+=("$(kubeedge::golang::get_target_by_binary $arg)")
  done

  if [[ ${#targets[@]} -eq 0 ]]; then
    for bin in ${KUBEEDGE_ALL_SMALL_BINARIES[@]}; do
        targets+=("$(kubeedge::golang::get_target_by_binary $bin)")
    done
  fi

  local -a binaries
  while IFS="" read -r binary; do binaries+=("$binary"); done < <(kubeedge::golang::binaries_from_targets "${targets[@]}")

  local ldflags
  read -r ldflags <<< "$(kubeedge::version::ldflags)"

  mkdir -p ${KUBEEDGE_OUTPUT_BINPATH}
  for bin in ${binaries[@]}; do
    echo "small building $bin"
    local name="${bin##*/}"
    set -x
    go build -o ${KUBEEDGE_OUTPUT_BINPATH}/${name} -ldflags "-w -s -extldflags -static $ldflags" $bin
    upx-ucl -9 ${KUBEEDGE_OUTPUT_BINPATH}/${name}
    set +x
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
	    -name '*_test.go' -print | xargs -n1 dirname | uniq)
    dirArray=(${findDirs// /})
    echo "${dirArray[@]}"
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

kubeedge::golang::get_pkg_test_dirs() {
    cd ${KUBEEDGE_ROOT}
    findDirs=$(find -L ./pkg \
	    -name '*_test.go' -print | xargs -n1 dirname | uniq)
    dirArray=(${findDirs// /})
    echo "${dirArray[@]}"
}

read -ra KUBEEDGE_CLOUD_TESTCASES <<< "$(kubeedge::golang::get_cloud_test_dirs)"
read -ra KUBEEDGE_EDGE_TESTCASES <<< "$(kubeedge::golang::get_edge_test_dirs)"
read -ra KUBEEDGE_KEADM_TESTCASES <<< "$(kubeedge::golang::get_keadm_test_dirs)"
read -ra KUBEEDGE_PKG_TESTCASES <<< "$(kubeedge::golang::get_pkg_test_dirs)"

readonly KUBEEDGE_ALL_TESTCASES=(
  ${KUBEEDGE_CLOUD_TESTCASES[@]}
  ${KUBEEDGE_EDGE_TESTCASES[@]}
  ${KUBEEDGE_KEADM_TESTCASES[@]}
  ${KUBEEDGE_PKG_TESTCASES[@]}
)

ALL_COMPONENTS_AND_GETTESTDIRS_FUNCTIONS=(
  cloud::::kubeedge::golang::get_cloud_test_dirs
  edge::::kubeedge::golang::get_edge_test_dirs
  keadm::::kubeedge::golang::get_keadm_test_dirs
  pkg::::kubeedge::golang::get_pkg_test_dirs
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

  local profile=${PROFILE:-""}
  if [[ $profile ]]; then
    go test "-coverprofile=${profile}" ${testdirs[@]}
    go tool cover -func=${profile}
  else
    go test ${testdirs[@]}
  fi
}
