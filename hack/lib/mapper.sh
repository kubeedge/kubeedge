#!/usr/bin/env bash

# Copyright 2020 The KubeEdge Authors.
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


set -o errexit
set -o nounset
set -o pipefail

kubeedge::mapper::build_bluetoothdevice() {
  blutooth_binary=${KUBEEDGE_GO_PACKAGE}/mappers/bluetooth_mapper
  name=bluetooth_mapper

  mkdir -p ${KUBEEDGE_OUTPUT_BINPATH}
  set -x
  go build -o ${KUBEEDGE_OUTPUT_BINPATH}/${name} $blutooth_binary
  set +x
}

kubeedge::mapper::build_modbusmapper() {
  modbus_binary=${KUBEEDGE_GO_PACKAGE}/mappers/modbus-go
  name=modbus

  mkdir -p ${KUBEEDGE_OUTPUT_BINPATH}
  set -x
  go build -o mappers/modbus-go/${name} $modbus_binary
  set +x
}
