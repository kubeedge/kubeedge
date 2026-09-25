#!/bin/bash -ex

# Copyright 2025 The KubeEdge Authors.
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

curpath=$PWD
echo $PWD
CONTAINER_RUNTIME=${CONTAINER_RUNTIME:-"containerd"}
mapper_image=modbus-e2e-mapper:v1.0.0

# build mapper project
cd ${curpath}/staging/src/github.com/kubeedge/mapper-framework
make generate modbus nostream && echo "successfully generated mapper project"

# copy device driver and add go work
cd ${curpath} && git clone https://github.com/wbc6080/modbus.git
cp -r ${curpath}/modbus/driver/* ${curpath}/staging/src/github.com/kubeedge/modbus/driver/
cp -r ${curpath}/modbus/config.yaml ${curpath}/staging/src/github.com/kubeedge/modbus/

# add encoding/json import and append AnomalyDetectionRequest struct to devicetype.go
sed -i '/^import (/a\	"encoding/json"' ${curpath}/staging/src/github.com/kubeedge/modbus/driver/devicetype.go
cat >> ${curpath}/staging/src/github.com/kubeedge/modbus/driver/devicetype.go << 'EOF'

type AnomalyDetectionRequest struct {
	Enabled                bool            `json:"enabled"`
	VisitorConfig          VisitorConfig   `json:"visitorConfig"`
	AnomalyDetectionConfig json.RawMessage `json:"anomalyDetectionConfig"`
	Data                   interface{}     `json:"data"`
}
EOF

# append AnomalyDetectionProcess function to driver.go
cat >> ${curpath}/staging/src/github.com/kubeedge/modbus/driver/driver.go << 'EOF'

func (c *CustomizedClient) AnomalyDetectionProcess(req *AnomalyDetectionRequest) error {
    // TODO: add the code to process anomaly detection
    return nil
}
EOF

go work use ./staging/src/github.com/kubeedge/modbus

# build modbus mapper image
cd ${curpath}/staging/src/github.com/kubeedge/modbus
CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go && sed -i '/go build/d' Dockerfile_nostream
docker build -f Dockerfile_nostream -t ${mapper_image} . && echo "successfully build test mapper image"

# import images to container-runtime
docker save -o modbus-mapper.tar ${mapper_image}

if [[ "${CONTAINER_RUNTIME}" = "cri-o" ]]; then
  # CRI-O resolves the short image name used by the e2e deployment to
  # docker.io/library/<name>, so the image has to be stored under that name.
  # A podman tag with a short target name stores localhost/<name> instead,
  # which CRI-O never finds, so always tag with the fully qualified name.
  crio_mapper_image=docker.io/library/${mapper_image}
  # Depending on the podman release, the loaded archive is named either after
  # the repository:tag it was saved with or localhost/v1.0.0:latest, so take
  # the name from the load output.
  load_output=$(sudo podman load -i modbus-mapper.tar)
  echo "${load_output}"
  loaded_image=$(printf '%s\n' "${load_output}" | awk -F': ' '/^Loaded image/ {print $2}' | tail -n 1)
  if [[ -z "${loaded_image}" ]]; then
    echo "failed to detect loaded modbus mapper image name"
    exit 1
  fi
  if [[ "${loaded_image}" != "${crio_mapper_image}" ]]; then
    sudo podman tag "${loaded_image}" "${crio_mapper_image}"
  fi
  if ! sudo podman image exists "${crio_mapper_image}"; then
    echo "modbus mapper image ${crio_mapper_image} not found in CRI-O storage"
    exit 1
  fi
  echo "successfully import modbus mapper image to CRI-O"
elif [[ "${CONTAINER_RUNTIME}" = "isulad" ]]; then
  sudo isula load -i modbus-mapper.tar && sudo isula tag modbus-e2e-mapper:v1.0.0 docker.io/library/modbus-e2e-mapper:v1.0.0 || true
  echo "successfully import modbus mapper image to Isulad"
elif [[ "${CONTAINER_RUNTIME}" = "containerd" ]]; then
  sudo ctr -n k8s.io images import modbus-mapper.tar && echo "successfully import modbus mapper image to Containerd"
elif [[ "${CONTAINER_RUNTIME}" = "docker" ]]; then
  echo "no need to import modbus mapper image"
else
  echo "not supported container runtime ${CONTAINER_RUNTIME}"
  exit 1
fi
