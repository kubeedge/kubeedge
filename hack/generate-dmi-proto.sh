#!/usr/bin/env bash
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

set -o errexit
set -o nounset
set -o pipefail

DMI_DIR=pkg/apis/dmi
DMI_VERSION=v1alpha1
DMI_API_FILE=api.proto
DMI_API_GO_FILE=api.pb.go

# try to parse named parameters
while [ $# -gt 0 ]; do
  case "$1" in
    --DMI_VERSION=*)
      DMI_VERSION="${1#*=}"
      ;;
    *)
      printf "***************************\n"
      printf "* Error: Invalid argument.*\n"
      printf "***************************\n"
      exit 1
  esac
  shift
done

DMI_CLIENT_CODE='
func NewMapperClient(cc grpc.ClientConnInterface) DeviceMapperServiceClient {
	return &deviceMapperServiceClient{cc}
}

func NewDeviceManageClient(cc grpc.ClientConnInterface) DeviceManagerServiceClient {
	return &deviceManagerServiceClient{cc}
}
'

# shellcheck disable=SC1004
COPY_RIGHT_INFO_LINE_1='/*\nCopyright 2022 The KubeEdge Authors.\n'
COPY_RIGHT_INFO_LINE_2='Licensed under the Apache License, Version 2.0 (the "License");'
COPY_RIGHT_INFO_LINE_3='you may not use this file except in compliance with the License.'
COPY_RIGHT_INFO_LINE_4='You may obtain a copy of the License at\n'
COPY_RIGHT_INFO_LINE_5='     http://www.apache.org/licenses/LICENSE-2.0\n'
COPY_RIGHT_INFO_LINE_6='Unless required by applicable law or agreed to in writing, software'
COPY_RIGHT_INFO_LINE_7='distributed under the License is distributed on an "AS IS" BASIS,'
COPY_RIGHT_INFO_LINE_8='WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.'
COPY_RIGHT_INFO_LINE_9='See the License for the specific language governing permissions and'
COPY_RIGHT_INFO_LINE_10='limitations under the License.\n*/\n'


cd ${DMI_DIR}
protoc -I "${DMI_VERSION}"/ --go_out=plugins=grpc:"${DMI_VERSION}" "${DMI_VERSION}/${DMI_API_FILE}"

sed -i '2,15d' "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_10"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_9"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_8"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_7"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_6"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_5"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_4"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_3"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_2"  "${DMI_VERSION}/${DMI_API_GO_FILE}"
sed -i "1i\ $COPY_RIGHT_INFO_LINE_1"  "${DMI_VERSION}/${DMI_API_GO_FILE}"

echo "${DMI_CLIENT_CODE}" >> "${DMI_VERSION}/${DMI_API_GO_FILE}"

gofmt -w "${DMI_VERSION}/${DMI_API_GO_FILE}"

echo "success to generate dmi in ${DMI_DIR}/${DMI_VERSION}/${DMI_API_GO_FILE}"
