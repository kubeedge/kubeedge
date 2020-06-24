#!/bin/bash

# Copyright 2019 The KubeEdge Authors.

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

echo 'create edgemesh init Container image'

function usage() {
	echo "execute 'sh createImg.sh [rpm | deb]' to create docker image"
	echo "execute 'sh createImg.sh help for use help'"
}

path="${1}"

if [ "${path}" != "rpm" ] && [ "${path}" != "deb" ]; then
	usage
	exit 0
fi

echo "create a ${path} docker image"

cp ./script/edgemesh-iptables.sh ./"${path}"/

cd ./"${path}"/

chmod 0777 edgemesh-iptables.sh

if command -v docker > /dev/null 2>&1 ; then
	#docker build
	docker build -t edgemesh_init .
	# delete iptables script
	rm ./edgemesh-iptables.sh
else
	echo 'the docker command is no found!!'
	exit 1
fi
