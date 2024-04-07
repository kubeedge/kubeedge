#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CURR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
# The root of the octopus directory
ROOT_DIR="${CURR_DIR}"
MAPPER_DIR="$(cd "$(dirname "$ROOT_DIR")" && pwd -P)"

function entry() {
  # copy template
  if [ $# -ne 2 ] ;then
    read -p "Please input the mapper name (like 'Bluetooth', 'BLE'): " -r mapperName
    if [[ -z "${mapperName}" ]]; then
      echo "the mapper name is required"
      exit 1
    fi
    read -p "Please input the build method (like 'stream', 'nostream'): " -r buildMethod
    if [[ -z "${buildMethod}" ]]; then
          echo "the build method is required"
          exit 1
    fi
  else
    mapperName=$1
    buildMethod=$2
  fi
  mapperNameLowercase=$(echo -n "${mapperName}" | tr '[:upper:]' '[:lower:]')
  mapperPath="${MAPPER_DIR}/${mapperNameLowercase}"
  if [[ -d "${mapperPath}" ]]; then
    echo "the directory is existed"
    exit 1
  fi
  cp -r "${ROOT_DIR}/_template/mapper" "${mapperPath}"
  if [ "${buildMethod}" = "stream" ]; then
    rm "${mapperPath}/data/stream/handler_nostream.go"
  fi

  if [ "${buildMethod}" = "nostream" ]; then
      cd "${mapperPath}/data/stream"
      ls |grep -v handler_nostream.go |xargs rm -rf
      mv handler_nostream.go handler.go
      cd -
  fi

  mapperVar=$(echo "${mapperName}" | sed -e "s/\b\(.\)/\\u\1/g")
  
  if [ $(uname) = "Darwin" ]; then
      sed -i "" "s/Template/${mapperVar}/g" `grep Template -rl ${mapperPath}`
      sed -i "" "s/kubeedge\/${mapperVar}/kubeedge\/${mapperNameLowercase}/g" `grep "kubeedge\/${mapperVar}" -rl $mapperPath`
  else
      sed -i "s/Template/${mapperVar}/g" `grep Template -rl ${mapperPath}`
      sed -i "s/kubeedge\/${mapperVar}/kubeedge\/${mapperNameLowercase}/g" `grep "kubeedge\/${mapperVar}" -rl $mapperPath`
  fi

  cd ${mapperPath} && go mod tidy

  empty_file_path="${MAPPER_DIR}/.empty"
  if [ -f "$empty_file_path" ]; then
      rm "$empty_file_path"
  fi
  echo "You can find your customized mapper in mappers "

}

entry "$@"