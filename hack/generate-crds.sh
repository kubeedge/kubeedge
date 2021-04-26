#!/usr/bin/env bash

# Copyright 2020 Authors of Arktos.
# Copyright 2020 The KubeEdge Authors - file modified.
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

crdVersions=v1
crdOutputs=build/crds
devicesVersion=v1alpha2
reliablesyncsVersion=v1alpha1

# install controller-gen tool if not exsit
if [ $(which controller-gen) == "" ]; then
    echo "Start to install controller-gen tool"
    GO111MODULE=on go get -v sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0
fi

# try to parse named parameters
while [ $# -gt 0 ]; do
  case "$1" in
    --crdVersions=*)
      crdVersions="${1#*=}"
      ;;
    --crdOutputs=*)
      crdOutputs="${1#*=}"
      ;;
    --devicesVersion=*)
      devicesVersion="${1#*=}"
      ;;
    --reliablesyncsVersion=*)
      reliablesyncsVersion="${1#*=}"
      ;;
    *)
      printf "***************************\n"
      printf "* Error: Invalid argument.*\n"
      printf "***************************\n"
      exit 1
  esac
  shift
done

# generate crds, grep the warnings from the std err, then print it. 
msg=$($(which controller-gen) paths="./..." crd:crdVersions=$crdVersions,allowDangerousTypes=true output:crd:artifacts:config=/tmp/crds  2> >(grep -i InterfaceType))
if  [ "$msg" != "" ]; then
    echo "The following warnings could be ignored, and the generated crds work well:"
    echo $msg | awk '{print "Warning:",$0}'   
fi

# rename files, copy files
mkdir -p ${crdOutputs}/devices
mkdir -p ${crdOutputs}/reliablesyncs

for entry in `ls /tmp/crds/*.yaml`; do
    crdName=$(echo ${entry} | cut -d'.' -f3 | cut -d'_' -f2)

    if [ "$crdName" == "devices" ] || [ "$crdName" == "devicemodels" ]; then
        # remove the last element if it is 's'
        if [ "${crdName: -1}" == "s" ]; then
          crdName="${crdName:$i:-1}"
        fi
        cp -v ${entry} ${crdOutputs}/devices/devices_${devicesVersion}_${crdName}.yaml 
    elif [ "$crdName" == "clusterobjectsyncs" ]; then
        cp -v ${entry} ${crdOutputs}/reliablesyncs/cluster_objectsync_${reliablesyncsVersion}.yaml
    elif [ "$crdName" == "objectsyncs" ]; then
        cp -v ${entry} ${crdOutputs}/reliablesyncs/objectsync_${reliablesyncsVersion}.yaml
    else
        # other cases would not handle
        continue
    fi
done

# clean
rm -rf /tmp/crds

