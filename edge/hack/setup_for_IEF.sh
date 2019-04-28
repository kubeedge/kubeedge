#!/usr/bin/env bash

# Copyright 2018 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if [ "$1" = "" ];then
    echo "Please specify the node configuration file"
    exist 1
fi

NODE_CONFIG_FILE=$1
SRC_DIR=${GOPATH}/src/github.com/kubeedge/kubeedge/edge

if [ -z "${GOPATH}" ]; then
    echo "Please export GOPATH"
    exit 1
fi

echo "make new dir for certs..."
if [ ! -d kubeedge_work_dir ]; then
	mkdir kubeedge_work_dir
else
	rm -rf kubeedge_work_dir/*
fi
KUBEEDGE_WORK_DIR=`pwd`/kubeedge_work_dir

echo "unzip to get the certificates and user_config..."
tar -zxvf ${NODE_CONFIG_FILE} -C kubeedge_work_dir/

CURRENT_PATH=${SRC_DIR}
CERT_PATH=${KUBEEDGE_WORK_DIR}
DEFAULT_PLACEMENT_FLAG="true"
create_system_config() {
    #docker root dir
set +e
    docker_info=`docker info |grep "Docker Root Dir"`
    local rt=$?
set -e
    if [ ${rt} -ne 0 ]; then
        echo "Docker Root Dir should be exist, please check docker version(>=1.11)"
        exit 1
    fi
    docker_root_dir=`echo $docker_info|sed 's/"//g' | sed 's/Docker Root Dir://'|sed 's/ //g'`
    if [ ! -f ${CURRENT_PATH}/conf/system.yaml ]; then
        cat > ${CURRENT_PATH}/conf/system.yaml <<EOF
systeminfo:
    architecture: {ARCH}
    docker_root_dir: ${docker_root_dir}
EOF
    else
        sed -i "s|{DOCKER_ROOT}|$docker_root_dir|g" ${CURRENT_PATH}/conf/system.yaml
    fi

    #ARCH
    #TODO: need to confirm the output of `arch` in different architecture：i386、i486、i586、alpha、sparc、arm、m68k、mips、ppc、i686 x86_64
    arch_info=`arch`
    case $arch_info in
    "x86_64" | "amd64")
        archInfo="x86_64"
        sed -i "s/{ARCH}/${arch_info}/g" ${CURRENT_PATH}/conf/system.yaml
        ;;
    "arm" | "armv7" | "armv7l")
        archInfo="arm32"
        sed -i "s/{ARCH}/${archInfo}/g" ${CURRENT_PATH}/conf/system.yaml
        ;;
    "aarch64")
        archInfo="arm64"
        sed -i "s/{ARCH}/${archInfo}/g" ${CURRENT_PATH}/conf/system.yaml
        ;;
    "i386")
        archInfo="i386"
        sed -i "s/{ARCH}/${archInfo}/g" ${CURRENT_PATH}/conf/system.yaml
        ;;
    *)
        echo "Don't support architecture ${arch_info}!"
        exit 1
        ;;
    esac
    export GOARCHAIUS_CONFIG_PATH=${CURRENT_PATH}
}

parse_config() {
    if [ ! -f ${CERT_PATH}/user_config ]; then
        echo "file user_config must be exist in the directory: ${CERT_PATH}"
        exit 1
    fi

    MASTER_ADDR_FOREDGE=`cat ${CERT_PATH}/user_config | grep -Po '"MASTER_URL":".*?"' | sed 's/"//g' | sed 's/MASTER_URL://'`
    if [ $? -ne 0 ] || [ "${MASTER_ADDR_FOREDGE}"x = x ]; then
        echo "Parse MASTER_URL failed!"
        exit 1
    fi

    NODE_HOST_NAME=`cat ${CERT_PATH}/user_config | grep -Po '"NODE_ID":".*?"' | sed 's/"//g' | cut -d":" -f2`
    if [ $? -ne 0 ] || [ "${NODE_HOST_NAME}"x = x ]; then
        echo "Parse NODE_ID failed!"
        exit 1
    fi

    EDGE_NAMESPACE=`cat ${CERT_PATH}/user_config | grep -Po '"PROJECT_ID":".*?"' | sed 's/"//g' | cut -d":" -f2`
    if [ $? -ne 0 ] || [ "${EDGE_NAMESPACE}"x = x ]; then
        echo "Parse PROJECT_ID failed!"
        exit 1
    fi

    PRIVATE_CERT_FILE=`cat ${CERT_PATH}/user_config | grep -Po '"PRIVATE_CERTIFICATE":".*?"' | sed 's/"//g' | sed 's/PRIVATE_CERTIFICATE://'`
    if [ $? -ne 0 ] || [ "${PRIVATE_CERT_FILE}"x = x ]; then
        echo "Parse PRIVATE_CERTIFICATE failed!"
        exit 1
    fi

    PRIVATE_KEY_FILE=`cat ${CERT_PATH}/user_config | grep -Po '"PRIVATE_KEY":".*?"' | sed 's/"//g' | sed 's/PRIVATE_KEY://'`
    if [ $? -ne 0 ] || [ "${PRIVATE_KEY_FILE}"x = x ]; then
        echo "Parse PRIVATE_KEY failed!"
        exit 1
    fi

    ROOT_CA_FILE=`cat ${CERT_PATH}/user_config | grep -Po '"ROOT_CA":".*?"' | sed 's/"//g' | sed 's/ROOT_CA://'`
    if [ $? -ne 0 ] || [ "${ROOT_CA_FILE}"x = x ]; then
        echo "Parse ROOT_CA failed!"
        exit 1
    fi

    ENABLE_GPU=`cat ${CERT_PATH}/user_config | grep -Po '"ENABLE_GPU":".*?"' | sed 's/"//g' | sed 's/ENABLE_GPU://'`
    if [ $? -ne 0 ] || [ "${ENABLE_GPU}"x = x ]; then
        echo "Parse ENABLE_GPU failed!"
        exit 1
    fi

    DIS_URL=`cat ${CERT_PATH}/user_config | grep -Po '"DIS_URL":".*?"' | sed 's/"//g' | sed 's/DIS_URL://'`
    if [ $? -ne 0 ] || [ "${DIS_URL}"x = x ]; then
        echo "Parse DIS_URL failed!"
        exit 1
    fi

    DIS_API_VERSION=`cat ${CERT_PATH}/user_config | grep -Po '"DIS_API_VERSION":".*?"' | sed 's/"//g' | sed 's/DIS_API_VERSION://'`
    if [ $? -ne 0 ] || [ "${DIS_API_VERSION}"x = x ]; then
        echo "Parse DIS_API_VERSION failed!"
        exit 1
    fi

    REGION=`cat ${CERT_PATH}/user_config | grep -Po '"REGION":".*?"' | sed 's/"//g' | sed 's/REGION://'`
    if [ $? -ne 0 ] || [ "${REGION}"x = x ]; then
        echo "Parse REGION failed!"
        exit 1
    fi

    OBS_URL=`cat ${CERT_PATH}/user_config | grep -Po '"OBS_URL":".*?"' | sed 's/"//g' | sed 's/OBS_URL://'`
    if [ $? -ne 0 ] || [ "${OBS_URL}"x = x ]; then
        echo "Parse OBS_URL failed!"
        exit 1
    fi
}

create_edge_config() {
    if [ ! -f ${CURRENT_PATH}/conf/edge.yaml ]; then
        echo "There is no ${CURRENT_PATH}/conf/edge.yaml!"
        exit 1
    fi
    sed -i "s|certfile: .*|certfile: ${CERT_PATH}/${PRIVATE_CERT_FILE}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|keyfile: .*|keyfile: ${CERT_PATH}/${PRIVATE_KEY_FILE}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|project-id: .*|project-id: ${EDGE_NAMESPACE}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|node-id: .*|node-id: ${NODE_HOST_NAME}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|placement: .*|placement: ${DEFAULT_PLACEMENT_FLAG}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|register-node-namespace: .*|register-node-namespace: ${EDGE_NAMESPACE}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|hostname-override: .*|hostname-override: ${NODE_HOST_NAME}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|device-plugin-enabled: .*|device-plugin-enabled: ${ENABLE_GPU}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|gpu-plugin-enabled: .*|gpu-plugin-enabled: ${ENABLE_GPU}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|placement-url: .*|placement-url: ${MASTER_ADDR_FOREDGE}/v1/placement_external/message_queue|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|api-version: .*|api-version: ${DIS_API_VERSION}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|region: .*|region: ${REGION}|g" ${CURRENT_PATH}/conf/edge.yaml
    sed -i "s|obs_endpoint: .*|obs_endpoint: ${OBS_URL}|g" ${CURRENT_PATH}/conf/edge.yaml
}


create_system_config
parse_config
create_edge_config
