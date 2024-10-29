#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

Template_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

MAPPER_DIR="$(cd "$(dirname "$Template_DIR")" && pwd -P)"

function entry() {
  if [ $# -ne 1 ] ;then
    mapperName="mapper_default"
  else
    mapperName=$1
  fi

  mapperPath="${MAPPER_DIR}/${mapperName}"

  if [[ -d "${mapperPath}" ]]; then
    echo "${mapperPath} is existed"
    exit 1
  fi
  mkdir -p "${mapperPath}/src"

  cp -r "${Template_DIR}/src" "${mapperPath}"
  cp -r "${Template_DIR}/pom.xml" "${mapperPath}"
  cp -r "${Template_DIR}/Dockerfile" "${mapperPath}"
  cp -r "${Template_DIR}/Makefile" "${mapperPath}"
  cp -r "${Template_DIR}/hack" "${mapperPath}"

  echo "Mapper has been created successfully, and the path is ${mapperPath}"
}

entry "$@"