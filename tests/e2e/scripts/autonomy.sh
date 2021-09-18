#!/bin/bash

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

workdir=`pwd`
cd $workdir

rootPath=$(cd ../../.. && pwd)
echo $rootPath

source ${rootPath}/tests/e2e/scripts/util.sh

checkContainerExist() {
  echo "wait for pod restart automatically..."
  for ((integer = 1; integer <= 20; integer++))
  do
    num=$(docker ps | grep nginx_autonomy | wc -l)
    if [[ "$num" == "0" ]]; then
      sleep 10
    else
      echo "found autonomy pod restart automatically"
      docker ps | grep nginx_autonomy
      break
    fi
  done

  if [[ "$num" == "0" ]];then
    echo "Edge Autonomy test has failures, pod doesn't restart when edgecore restart. Please check !!"
    exit 1
  else
    echo "Edge Autonomy test successfully passed all the tests !!"
  fi
}

start_edgecore() {
  EDGE_CONFIGFILE=${rootPath}/_output/local/bin/edgecore.yaml
  EDGE_BIN=${rootPath}/_output/local/bin/edgecore
  EDGECORE_LOG="/tmp/edgecore.log"

  echo "start edgecore..."
  export CHECK_EDGECORE_ENVIRONMENT="false"
  nohup sudo -E ${EDGE_BIN} --config=${EDGE_CONFIGFILE} > "${EDGECORE_LOG}" 2>&1 &
  EDGECORE_PID=$!
  echo $EDGECORE_PID
}

testAutonomy() {
  # print docker progresses for user debug.
  echo "now we have autonomy test pod as follows."
  docker ps --filter "name=nginx_autonomy"

  # pkill edgecore
  kill_component "edgecore"

  # pkill cloudcore
  kill_component "cloudcore"

  # kill all docker container progresses
  for id in $(docker ps --filter "name=nginx_autonomy" -q)
  do
    docker ps --filter "name=nginx_autonomy"
    docker kill $id
  done

  # restart edgecore
  start_edgecore

  # check whether containers restart automatically and successfully
  checkContainerExist

  # ensure all the related containers stopped.
  for id in $(docker ps --filter "name=nginx_autonomy" -q)
  do
    docker ps --filter "name=nginx_autonomy"
    docker kill $id
  done
}

testAutonomy
