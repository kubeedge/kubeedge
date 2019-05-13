#!/usr/bin/env bash
# Copyright 2019 The KubeEdge Authors.
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

SRC_DIR=${GOPATH}/src/github.com/kubeedge/kubeedge

imageRepo=$1

if [ ! -z $(docker images -q ${imageRepo}/edgecore:latest) ];then
    echo "Image exist locally !!"
    docker rmi -f $(docker images -q ${imageRepo}/edgecore:latest)
fi

cd ${SRC_DIR}

docker build -t ${imageRepo}/edgecore:latest -f ${SRC_DIR}/build/edge/Dockerfile .

docker push ${imageRepo}/edgecore:latest

echo "edgecore image successully built and pushed to repository !!"
