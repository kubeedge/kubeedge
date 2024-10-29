#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

curr_project="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
mapperName="$(basename "${curr_project}")"

function entry() {
  if [ $# -ne 1 ] ;then
    imageName="$(basename "${curr_project}")"
  else
    imageName=$1
  fi

  docker build -t "${imageName}" --build-arg mapperName="${mapperName}" .
  echo "Docker image: ${imageName} build successfully"
}

entry "$@"