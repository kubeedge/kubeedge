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
  if [ $# -eq 0 ] ;then
    read -p "Please input the mapper name (like 'Bluetooth', 'BLE'): " -r mapperName
    if [[ -z "${mapperName}" ]]; then
      echo "the mapper name is required"
      exit 1
    fi
  else
    mapperName=$1
  fi
  mapperNameLowercase=$(echo -n "${mapperName}" | tr '[:upper:]' '[:lower:]')
  mapperPath="${MAPPER_DIR}/${mapperNameLowercase}"
  if [[ -d "${mapperPath}" ]]; then
    echo "the directory is existed"
    exit 1
  fi
  cp -r "${ROOT_DIR}/_template/mapper" "${mapperPath}"

  mapperVar=$(echo "${mapperName}" | sed -e "s/\b\(.\)/\\u\1/g")
  sed -i "s/Template/${mapperVar}/g" `grep Template -rl ${mapperPath}`
  sed -i "s/kubeedge\/${mapperVar}/kubeedge\/${mapperNameLowercase}/g" `grep "kubeedge\/${mapperVar}" -rl $mapperPath`

  cd ${mapperPath} && go mod tidy

  empty_file_path="${MAPPER_DIR}/.empty"
  if [ -f "$empty_file_path" ]; then
      rm "$empty_file_path"
  fi
  echo "You can find your customized mapper in mappers "

}

entry "$@"