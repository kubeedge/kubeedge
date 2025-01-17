#!/usr/bin/env bash

###
#Copyright 2019 The KubeEdge Authors.
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

# The root of the build/dist directory
KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "${KUBEEDGE_ROOT}/hack/lib/init.sh"

# === Capture go / godebug directives from root go.mod
go_directive_value=$(grep '^go 1.' go.mod | awk '{print $2}' || true)
if [[ -z "${go_directive_value}" ]]; then
  echo "root go.mod must have 'go 1.x.y' directive"
  exit 1
fi

# update go work
(
  echo "running update go work"
  unset GOWORK
  unset GOFLAGS
  if [[ ! -f go.work ]]; then
    echo "go.work: initialize"
    go work init
  fi
  # Prune use directives
  go work edit -json \
      | jq -r '.Use[]? | "-dropuse \(.DiskPath)"' \
      | xargs -L 100 go work edit -fmt
  # Ensure go and godebug directives
  go work edit -go "${go_directive_value}"
  # Re-add use directives
  go work use .
  for repo in $(kubeedge::util::list_staging_repos); do
    go work use "./staging/src/github.com/kubeedge/${repo}"
  done
  go work sync
)

# update go.mod and go.sum for staging repos
for repo in $(kubeedge::util::list_staging_repos); do
  pushd "${KUBEEDGE_ROOT}/staging/src/github.com/kubeedge/${repo}"
  echo "running 'go mod tidy' for ${repo}"
  go mod tidy

  # go mod tidy sometimes removes lines that build seems to need. See also https://github.com/golang/go/issues/31248.
  # We would have to always execute go mod vendor after go mod tidy to ensure correctness.
  echo "running 'go mod vendor' for ${repo}"
  GOWORK=off go mod vendor

  # vendor/ is not supposed to exist in staging repos, remove it.
  rm -rf vendor/

  popd
done


echo "running 'go mod tidy' for repo root"
go mod tidy

echo "running 'go mod vendor' for repo root"
GOWORK=off go mod vendor

# create a symlink in vendor directory pointing to the staging components.
# This lets other packages and tools use the local staging components as if they were vendored.
for repo in $(kubeedge::util::list_staging_repos); do
  rm -fr "${KUBEEDGE_ROOT}/vendor/github.com/kubeedge/${repo}"
  ln -s "../../../staging/src/github.com/kubeedge/${repo}/" "${KUBEEDGE_ROOT}/vendor/github.com/kubeedge/${repo}"
done
