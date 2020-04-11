#!/usr/bin/env bash

###
#Copyright 2020 The KubeEdge Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
#
#This script does the preparation work for the Debian Packages to be created.
# It accepts two parameters ARCH and GIT_TAG

# The root of the build/dist directory
KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
# The temporary location to store files
TEMP_LOC="/tmp/"
ARCH=$1
GIT_TAG=$2

# Creating folders to store files for installation for edgecore
mkdir -p "${TEMP_LOC}/edgecore/edgecore_folder"
# Copying files from the github repo to the respective folders
# Creating the post_install script for Edgecore
cp "${KUBEEDGE_ROOT}/build/tools/edgecore.service" "$TEMP_LOC/edgecore/edgecore_folder/edgecore.service"
echo "mkdir -p  /etc/kubeedge/config/" >> $TEMP_LOC/edgecore/edgecore_folder/post_install.sh
echo "/usr/bin/kubeedge/edgecore --minconfig > /etc/kubeedge/config/edgecore.yaml" >> $TEMP_LOC/edgecore/edgecore_folder/post_install.sh

# Copying the cloudcore and edgecore build for arm64 to respective folders
# Binaries are copied into $TEMP_LOC/cloudcore and $TEMP_LOC/edgecore folder
cp "${KUBEEDGE_ROOT}/_output/local/bin/edgecore" "$TEMP_LOC/edgecore/edgecore"

# Creating Packages for cloudcore/ edgecore for arm64. -v <version> needs to be replaced with tag version.
mkdir -p "${KUBEEDGE_ROOT}/_output/local/pkg"
cd "${KUBEEDGE_ROOT}/_output/local/pkg" || exit
fpm -s dir -t deb -v $GIT_TAG -a $ARCH -n kubeedge-edgecore --after-install=$TEMP_LOC/edgecore/edgecore_folder/post_install.sh $TEMP_LOC/edgecore/edgecore_folder/=/etc/kubeedge/files $TEMP_LOC/edgecore/edgecore=/usr/bin/kubeedge/edgecore $TEMP_LOC/edgecore/edgecore_folder/edgecore.service=/etc/systemd/system/edgecore.service
