#!/bin/bash

# Copyright 2022 The KubeEdge Authors.
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

set -o pipefail

FROM_VERSION=${FROM_VERSION:-"v1.9.1"}
TO_VERSION=${TO_VERSION:-"v1.10.0"}

BACKUP_PATH=${BACKUP_PATH:-"/etc/kubeedge/backup/${FROM_VERSION}"}
UPGRADE_PATH=${UPGRADE_PATH:-"/etc/kubeedge/upgrade/${TO_VERSION}"}

DB_PATH=${DB_PATH:-"/var/lib/kubeedge/edgecore.db"}
EDGECORE_CONFIG_PATH=${EDGECORE_CONFIG_PATH:-"/etc/kubeedge/config/edgecore.yaml"}
EDGECORE_BIN_PATH=${EDGECORE_BIN_PATH:-"/usr/local/bin/edgecore"}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source ${ROOT_DIR}/common.sh

if [[ ${KUBEEDGE_LOCAL_UP} == "true" ]];then {
    DB_PATH="/tmp/var/lib/kubeedge/edgecore.db"
    EDGECORE_CONFIG_PATH="/root/go/src/github.com/kubeedge/kubeedge/_output/local/bin/edgecore.yaml"
}
fi

# ensure backup dir exist
mkdir -p ${BACKUP_PATH}
mkdir -p ${UPGRADE_PATH}

if [ ! -f ${DB_PATH} ]; then
    logger_Error "old package dir has no database"
    exit 1
else
    cp ${DB_PATH} ${BACKUP_PATH}/edgecore.db
fi

if [ ! -f ${EDGECORE_CONFIG_PATH} ]; then
    logger_Error "old package dir has no edgecore config"
    exit 1
else
    cp ${EDGECORE_CONFIG_PATH} ${BACKUP_PATH}/edgecore.yaml
fi

if [ ! -f ${EDGECORE_BIN_PATH} ]; then
    logger_Error "old package dir has no edgecore binary"
    exit 1
else
    cp ${EDGECORE_BIN_PATH} ${BACKUP_PATH}/edgecore
fi

# copy edgecore binary from docker image
IMAGE_NAME="kubeedge/installation-package:${TO_VERSION}"

output=$(docker pull ${IMAGE_NAME} 2>&1)
if [[ $? -eq 0 ]]; then
    logger_Info "docker pull image successfully " $output
else
    logger_Error "docker pull image failed " $output
    exit 1
fi

output=$(docker_id="$(docker create ${IMAGE_NAME})" && docker cp ${docker_id}:/usr/local/bin/edgecore ${UPGRADE_PATH}/edgecore && docker rm -v ${docker_id} 2>&1)

if [[ $? -eq 0 ]]; then
    logger_Info "docker copy edgecore successfully " $output
else
    logger_Error "docker copy edgecore failed " $output
    exit 1
fi


output=$(create_idempotency_dir)
if [[ $? -eq 0 ]]; then
    logger_Info "create idempotency path&file successfully" $output
else
    logger_Error "create idempotency path&file failed" $output
    exit 1
fi

check_idempotency_record
if [ $? -ne 0 ]; then
    logger_Error "There is something failed when upgraded last time, please check the log and node manually."
    echo "There is something failed when upgraded last time, please check the log and node manually."
    exit 1
fi
