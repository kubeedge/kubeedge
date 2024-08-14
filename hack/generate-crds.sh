#!/usr/bin/env bash

# Copyright 2021 The KubeEdge Authors 
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

CRD_VERSIONS=v1
CRD_OUTPUTS=build/crds
DEVICES_VERSION=v1beta1
OPERATIONS_VERSION=v1alpha1
RELIABLESYNCS_VERSION=v1alpha1
SERVICEACCOUNTACCESS_VERSION=v1alpha1
APPS_VERSION=v1alpha1
HELM_CRDS_DIR=manifests/charts/cloudcore/crds
ROUTER_DIR=build/crds/router

_crdOptions="crd:crdVersions=${CRD_VERSIONS},generateEmbeddedObjectMeta=true,allowDangerousTypes=true"
_tmpdir=/tmp/crds

# try to parse named parameters
while [ $# -gt 0 ]; do
  case "$1" in
    --CRD_VERSIONS=*)
      CRD_VERSIONS="${1#*=}"
      ;;
    --CRD_OUTPUTS=*)
      CRD_OUTPUTS="${1#*=}"
      ;;
    --DEVICES_VERSION=*)
      DEVICES_VERSION="${1#*=}"
      ;;
    --OPERATIONS_VERSION=*)
      OPERATIONS_VERSION="${1#*=}"
      ;;
    --RELIABLESYNCS_VERSION=*)
      RELIABLESYNCS_VERSION="${1#*=}"
      ;;
    *)
      printf "***************************\n"
      printf "* Error: Invalid argument.*\n"
      printf "***************************\n"
      exit 1
  esac
  shift
done

function :pre:install: {
  # install controller-gen tool if not exist
  if [ "$(which controller-gen)" == "" ]; then
      echo "Start to install controller-gen tool"
      GO111MODULE=on go install -v sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0
      GOPATH="${GOPATH:-$(go env GOPATH)}"
      export PATH=$PATH:$GOPATH/bin
  fi

  # TODO: When Kubernetes fixes this issue, we can remove this.
  local install_osname=$(uname -s | tr '[:upper:]' '[:lower:]')
  local yq_url="https://github.com/mikefarah/yq/releases/download/v4.44.3/yq_${install_osname}_amd64"
  command -v yq >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    echo "not exists yq installing it..."
    wget $yq_url -O "/usr/local/bin/yq"
    chmod +x "/usr/local/bin/yq"
  fi
}

function :gen:crds: {
  # generate crds
  cd staging/src/github.com/kubeedge/api/apis
  $(which controller-gen) paths="./..." ${_crdOptions} output:crd:artifacts:config=${_tmpdir}
  # ServiceAccount.Secret.Name property is in x-kubernetes-list-map-keys, 
  # but it is not have a default and not be a required property.
  # So, we need to add it to required properties through yq.
  # TODO: When Kubernetes fixes this issue, we can remove this.
  yq -i '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.serviceAccount.properties.secrets.items.required[0]="name"' ${_tmpdir}/policy.kubeedge.io_serviceaccountaccesses.yaml
  cd -
}

# remove the last element if it ends with "s", i.e: devicemodels -> devicemodel
function remove_suffix_s {
  if [ "${1: -1}" == "s" ]; then
    echo ${1%?}
  fi
}

function :copy:to:destination {
  # rename files, copy files
  mkdir -p ${CRD_OUTPUTS}/devices
  mkdir -p ${CRD_OUTPUTS}/reliablesyncs
  mkdir -p ${CRD_OUTPUTS}/apps
  mkdir -p ${CRD_OUTPUTS}/policy
  mkdir -p ${CRD_OUTPUTS}/operations

  for entry in `ls /tmp/crds/*.yaml`; do
      CRD_NAME=$(echo ${entry} | cut -d'.' -f3 | cut -d'_' -f2)

      if [ "$CRD_NAME" == "devices" ] || [ "$CRD_NAME" == "devicemodels" ]; then
          CRD_NAME=$(remove_suffix_s "$CRD_NAME") 
          cp -v ${entry} ${CRD_OUTPUTS}/devices/devices_${DEVICES_VERSION}_${CRD_NAME}.yaml
          cp -v ${entry} ${HELM_CRDS_DIR}/devices_${DEVICES_VERSION}_${CRD_NAME}.yaml 
      elif [ "$CRD_NAME" == "edgeapplications" ] || [ "$CRD_NAME" == "nodegroups" ]; then
          CRD_NAME=$(remove_suffix_s "$CRD_NAME")
          cp -v ${entry} ${CRD_OUTPUTS}/apps/apps_${APPS_VERSION}_${CRD_NAME}.yaml
          cp -v ${entry} ${HELM_CRDS_DIR}/apps_${APPS_VERSION}_${CRD_NAME}.yaml
      elif [ "$CRD_NAME" == "serviceaccountaccesses" ]; then
          CRD_NAME="serviceaccountaccess"
          cp -v ${entry} ${CRD_OUTPUTS}/policy/policy_${SERVICEACCOUNTACCESS_VERSION}_${CRD_NAME}.yaml
          cp -v ${entry} ${HELM_CRDS_DIR}/policy_${SERVICEACCOUNTACCESS_VERSION}_${CRD_NAME}.yaml
      elif [ "$CRD_NAME" == "clusterobjectsyncs" ]; then
          cp -v ${entry} ${CRD_OUTPUTS}/reliablesyncs/cluster_objectsync_${RELIABLESYNCS_VERSION}.yaml
          cp -v ${entry} ${HELM_CRDS_DIR}/cluster_objectsync_${RELIABLESYNCS_VERSION}.yaml
      elif [ "$CRD_NAME" == "objectsyncs" ]; then
          cp -v ${entry} ${CRD_OUTPUTS}/reliablesyncs/objectsync_${RELIABLESYNCS_VERSION}.yaml
          cp -v ${entry} ${HELM_CRDS_DIR}/objectsync_${RELIABLESYNCS_VERSION}.yaml
      elif [ "$CRD_NAME" == "nodeupgradejobs" ] || [ "$CRD_NAME" == "imageprepulljobs" ]; then
          CRD_NAME=$(remove_suffix_s "$CRD_NAME")
          cp -v ${entry} ${CRD_OUTPUTS}/operations/operations_${OPERATIONS_VERSION}_${CRD_NAME}.yaml
          cp -v ${entry} ${HELM_CRDS_DIR}/operations_${OPERATIONS_VERSION}_${CRD_NAME}.yaml
      else
          # other cases would not handle
          continue
      fi
  done

  for r_entry in `ls ${ROUTER_DIR}/*.yaml`; do
    # cp router CRDs  
    cp -v ${r_entry} ${HELM_CRDS_DIR}/
  done
}

function cleanup {
  #echo "Removing templates files: ${_tmpdir}"
  rm -rf "${_tmpdir}"
}

:pre:install:

:gen:crds:

:copy:to:destination

cleanup
