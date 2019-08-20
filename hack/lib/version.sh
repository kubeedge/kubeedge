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

KUBEEDGE_GO_PACKAGE="github.com/kubeedge/kubeedge"

edge::version::get_version_info() {

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
edge::version::ldflags() {
  edge::version::get_version_info

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

edge::version::ldflags

