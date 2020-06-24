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

apt-get update
apt-get install -y apt-transport-https  ca-certificates curl gnupg2 software-properties-common
curl -fsSL https://download.docker.com/linux/raspbian/gpg | apt-key add -
echo "deb [arch=armhf]  https://download.docker.com/linux/raspbian stretch stable" | tee /etc/apt/sources.list.d/docker.list
apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io
