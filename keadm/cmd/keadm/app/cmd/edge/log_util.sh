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

EDGE_CORE_INSTALL_PATH=/var/log/kubeedge
EDGE_CORE_INSTALL_FILE=edge_installer_script.log
MODE_NAME="Edge-core"

date_format="+%Y-%m-%dT%T"

# Some useful colors.
# check if stdout is a terminal and support colors...
if [ -t 1 ] && [ "1$(tput colors 2>/dev/null)" -ge 18 ]; then
  readonly color_red="$(tput setaf 1)"
  readonly color_yellow="$(tput setaf 3)"
  readonly color_green="$(tput setaf 2)"
  readonly color_norm="$(tput sgr0)"
else
  readonly color_red=""
  readonly color_yellow=""
  readonly color_green=""
  readonly color_norm=""
fi

if command -v caller >/dev/null 2>&1; then
  # return func(lineno:filename)
  # NOTE: skip 2-level inner frame
  _caller() { caller 2| awk '{sub(/.*\//,e,$3);print $2"("$3":"$1") "}'; }
else
  _caller() { :; }
fi

if [ ! -d ${EDGE_CORE_INSTALL_PATH} ]; then
  mkdir -p ${EDGE_CORE_INSTALL_PATH} && chmod -R 750 ${EDGE_CORE_INSTALL_PATH}
fi
if [ ! -f ${EDGE_CORE_INSTALL_PATH}/${EDGE_CORE_INSTALL_FILE} ]; then
  touch ${EDGE_CORE_INSTALL_PATH}/${EDGE_CORE_INSTALL_FILE} && chmod 640 ${EDGE_CORE_INSTALL_PATH}/${EDGE_CORE_INSTALL_FILE}
fi

_log()
{
  level=$1
  shift 1
  echo "$(date ${date_format}) -${MODE_NAME}- ${level} $(_caller)- $*" >> ${EDGE_CORE_INSTALL_PATH}/${EDGE_CORE_INSTALL_FILE}
}

logger_Info()
{
  _log INFO "$@"
}

logger_Warn()
{
  _log WARN "${color_yellow}$*${color_norm}"
}

logger_Error()
{
  _log ERROR "${color_red}$*${color_norm}"
}

die()
{
  _log ERROR "${color_red}$*${color_norm}"
  exit 1
}
