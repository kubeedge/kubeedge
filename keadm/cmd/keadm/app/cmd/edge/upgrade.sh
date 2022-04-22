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

EDGE_CORE_SERVICE_FILE=${EDGE_CORE_SERVICE_FILE:-"edgecore.service"}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source ${ROOT_DIR}/common.sh

if [ ${KUBEEDGE_LOCAL_UP} == "true" ];then {
    DB_PATH="/tmp/var/lib/kubeedge/edgecore.db"
    EDGECORE_CONFIG_PATH="/root/go/src/github.com/kubeedge/kubeedge/_output/local/bin/edgecore.yaml"
}
fi

stop_service ${EDGE_CORE_SERVICE_FILE} || pkill edgecore
if [ $? -ne 0 ]; then
    logger_Error "stop edgecore service failed"
    logger_Error "upgrade failed"
    write_idempotency_record ${HANDLE_UPGRADE_FAILED}
    exit 1
else
    logger_Info "stop old edgecore.service successfully"
fi

# replace old edgecore with new edgecore
cp ${UPGRADE_PATH}/edgecore /usr/local/bin/edgecore

logger_Info "copy new edgecore to /usr/local/bin successfully"

start_service ${EDGE_CORE_SERVICE_FILE}
if [ $? -eq 0 ]; then
    logger_Info "Complete starting edge_core process"
else
    write_idempotency_record ${HANDLE_UPGRADE_FAILED}
    logger_Error "Failed to start edge_core process, please check"
    exit 1
fi

write_idempotency_record ${HANDLE_UPGRADE_FINISHED}
logger_Info "Congratulations, upgrade edgecore successfully."
