#!/usr/bin/env bash

###
#Copyright 2019 The KubeEdge Authors.
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

# Creating folders to store files for installation for cloudcore
mkdir -p "${TEMP_LOC}/cloudcore/cloudcore_folder"
# Copying files from the github repo to the respective folders
cp "${KUBEEDGE_ROOT}/build/tools/certgen.sh" "$TEMP_LOC/cloudcore/cloudcore_folder/certgen.sh"
cp ${KUBEEDGE_ROOT}/build/crds/devices/* $TEMP_LOC/cloudcore/cloudcore_folder/.
cp ${KUBEEDGE_ROOT}/build/crds/reliablesyncs/* $TEMP_LOC/cloudcore/cloudcore_folder/.
cp "${KUBEEDGE_ROOT}/build/tools/cloudcore.service" "$TEMP_LOC/cloudcore/cloudcore_folder/cloudcore.service"

# Creating the post_install script for Cloudcore
echo "/etc/kubeedge/files/certgen.sh genCertAndKey edge" >> $TEMP_LOC/cloudcore/cloudcore_folder/post_install.sh
echo "kubectl create -f devices_v1alpha1_devicemodel.yaml" >> $TEMP_LOC/cloudcore/cloudcore_folder/post_install.sh
echo "kubectl create -f devices_v1alpha1_device.yaml" >> $TEMP_LOC/cloudcore/cloudcore_folder/post_install.sh
echo "kubectl create -f cluster_objectsync_v1alpha1.yaml" >> $TEMP_LOC/cloudcore/cloudcore_folder/post_install.sh
echo "kubectl create -f objectsync_v1alpha1.yaml" >> $TEMP_LOC/cloudcore/cloudcore_folder/post_install.sh
echo "mkdir -p  /etc/kubeedge/config/" >> $TEMP_LOC/cloudcore/cloudcore_folder/post_install.sh
echo "/usr/local/bin/kubeedge/cloudcore --minconfig > /etc/kubeedge/config/cloudcore.yaml" >> $TEMP_LOC/cloudcore/cloudcore_folder/post_install.sh

# Copying the cloudcore and edgecore build for arm64 to respective folders
# Binaries are copied into $TEMP_LOC/cloudcore folder
cp "${KUBEEDGE_ROOT}/_output/local/bin/cloudcore" "$TEMP_LOC/cloudcore/cloudcore"

# Creating Packages for cloudcore for arm64. -v <version> needs to be replaced with tag version.
mkdir -p "${KUBEEDGE_ROOT}/_output/local/pkg"
cd "${KUBEEDGE_ROOT}/_output/local/pkg" || exit
fpm -s dir -t deb -v $GIT_TAG -a $ARCH -d kubectl -n kubeedge-cloudcore --after-install=$TEMP_LOC/cloudcore/cloudcore_folder/post_install.sh $TEMP_LOC/cloudcore/cloudcore_folder/=/etc/kubeedge/files $TEMP_LOC/cloudcore/cloudcore=/usr/local/bin/kubeedge/cloudcore $TEMP_LOC/cloudcore/cloudcore_folder/cloudcore.service=/etc/systemd/system/cloudcore.service
