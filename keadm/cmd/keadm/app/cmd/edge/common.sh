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

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source ${ROOT_DIR}/log_util.sh

stop_service() {
    SERVICE_FILE=$1
    if [ -f /etc/systemd/system/${SERVICE_FILE} ]; then
        systemctl stop ${SERVICE_FILE}  >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            logger_Error "systemctl stop ${SERVICE_FILE} with error, please check."
            return 1
        fi

        systemctl disable ${SERVICE_FILE}  >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            logger_Error "systemctl disable ${SERVICE_FILE} with error, please check."
            return 1
        fi
    fi
    if [ -f /etc/systemd/system/${SERVICE_FILE} ]; then
        rm -f /etc/systemd/system/${SERVICE_FILE}
    else
        logger_Error "systemctl no ${SERVICE_FILE}"
        return 1
    fi
}


SERVICE_TEMPLATE="[Unit]
Description=edgecore.service

[Service]
Type=simple
ExecStart=/usr/local/bin/edgecore
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
"

start_service() {
    SERVICE_FILE=$1

    if [ ! -d /etc/systemd/system ]; then
        logger_Error "Not found /etc/systemd/system/"
        return 1
    fi

    echo -e "${SERVICE_TEMPLATE}" > /etc/systemd/system/${SERVICE_FILE}
    systemctl daemon-reload >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        logger_Error "systemctl daemon-reload with error, please check."
        return 1
    fi
    systemctl enable ${SERVICE_FILE}  >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        logger_Error "systemctl enable ${SERVICE_FILE} with error, please check."
        return 1
    fi
    systemctl start ${SERVICE_FILE}  >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        logger_Error "systemctl start ${SERVICE_FILE} with error, please check."
        exit 1
    fi
}


TASK="/etc/kubeedge/task"
HANDLE_UPGRADE_FAILED="handle_upgrade_failed"
HANDLE_UPGRADE_FINISHED="handle_upgrade_finished"
IDEMPOTENCY_PATH="/etc/kubeedge"

write_idempotency_record() {
    idempotency_str=$1
    if [ -z "${idempotency_str+x}" ] || [ "${idempotency_str}" = "" ]; then
        return
    fi

    # if idempotency record already exist, no need to record it again
    get_idempotency_record ${idempotency_str}
    if [ $? -eq 1 ]; then
        return
    fi
    idempotency_file=`cat ${TASK} | grep "idempotency_file" | cut -d":" -f2` || true
    if [ -z "${idempotency_file+x}" ] || [ "${idempotency_file}" = "" ]; then
        logger_Error "No idempotency file"
        exit 1
    fi

    if [ ! -f ${idempotency_file} ]; then
        echo "There is no file ${idempotency_file}"
        exit 1
    fi
    echo "  ${idempotency_str}" >> ${idempotency_file}
}


get_idempotency_record() {
    idempotency_str=$1
    if [ -z "${idempotency_str+x}" ] || [ "${idempotency_str}" = "" ]; then
        return
    fi

    local idempotency_file=`cat ${TASK} | grep "idempotency_file" | cut -d":" -f2`
    if [ -z "${idempotency_file+x}" ] || [ "${idempotency_file}" = "" ]; then
        logger_Info "No idempotency file"
        return 2
    fi
    if [ ! -f ${idempotency_file} ]; then
        return 0
    fi

    result=`cat ${idempotency_file} | grep $idempotency_str$ | grep -v grep` || true

    if [ -z "${result+x}" ] || [ "${result}" = "" ]; then
        return 0
    else
        return 1
    fi
}


ageing_idempotency_file() {
    current_idempotency_file=$1

    get_idempotency_record ${HANDLE_UPGRADE_FINISHED}
    is_handle_upgrade_finished=$?
    if [ ${is_handle_upgrade_finished} -eq 1 ]; then
        time_stamp=`date +'%Y-%m-%d-%H-%M'`
        mv ${current_idempotency_file} ${current_idempotency_file}-${time_stamp}
    fi
}

# if HANDLE_UPGRADE_FAILED record exist, then report failure and exit, inform users to process it manually.
check_idempotency_record() {
    get_idempotency_record ${HANDLE_UPGRADE_FAILED}
    is_handle_upgrade_failed=$?

    if [ ${is_handle_upgrade_failed} -eq 1 ]; then
        return 1
    fi
}

create_idempotency_dir() {
    task="upgrade"
    if [ ! -d ${IDEMPOTENCY_PATH} ]; then
        mkdir -p ${IDEMPOTENCY_PATH} && chmod -R 700 ${IDEMPOTENCY_PATH}
    fi
    if [ ! -f ${IDEMPOTENCY_PATH}/task ]; then
        touch ${IDEMPOTENCY_PATH}/task && chmod 600 ${IDEMPOTENCY_PATH}/task
    fi

    old_version=${FROM_VERSION}
    new_version=${TO_VERSION}

    # if version are the same, look at task to check whether it already update, if so, then return commonly.
    if [ "${old_version}" = "${new_version}" ]; then
        version=`cat ${IDEMPOTENCY_PATH}/task | grep "new_version" | cut -d":" -f2`
        task=`cat ${IDEMPOTENCY_PATH}/task | grep "task" | cut -d":" -f2`
        if [ "${version}" = "${old_version}" ] && [ "${task}" = "upgrade" ]; then
            logger_Warn "The version ${new_version} had been upgrade on this edge before."
            return
        else
            echo "the ${new_version} should had been upgrade on this edge before, please check!"
            logger_Error "the ${new_version} should had been upgrade on this edge, please check!"
            exit 1
        fi
    fi

    echo "task:upgrade" > ${IDEMPOTENCY_PATH}/task
    echo "old_version:${old_version}" >> ${IDEMPOTENCY_PATH}/task
    echo "new_version:${new_version}" >> ${IDEMPOTENCY_PATH}/task

    idempotency_file=${old_version}_${new_version}.${task}
    echo "idempotency_file:${IDEMPOTENCY_PATH}/${idempotency_file}" >> ${IDEMPOTENCY_PATH}/task

    # ageing idempotency record
    if [ -f ${IDEMPOTENCY_PATH}/${idempotency_file} ]; then
        ageing_idempotency_file ${IDEMPOTENCY_PATH}/${idempotency_file} || true
    fi
    if [ ! -f ${IDEMPOTENCY_PATH}/${idempotency_file} ]; then
        touch ${IDEMPOTENCY_PATH}/${idempotency_file} && chmod 600 ${IDEMPOTENCY_PATH}/${idempotency_file}
        echo "old_version:${old_version}" > ${IDEMPOTENCY_PATH}/${idempotency_file}
        echo "new_version:${new_version}" >> ${IDEMPOTENCY_PATH}/${idempotency_file}
        echo "upgrade time: `date +'%Y-%m-%d %H:%M:%S'`" >> ${IDEMPOTENCY_PATH}/${idempotency_file}
        echo "process:"  >> ${IDEMPOTENCY_PATH}/${idempotency_file}
    fi
}
