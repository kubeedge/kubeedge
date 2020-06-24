#!/bin/sh

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

CONFIG_DIR=/opt/src/conf

for VAR in $(env)
do
    if [[ ! -z "$(echo $VAR | grep -E '^CONNECTOR_')" ]]; then
        VAR_NAME=$(echo "$VAR" | sed -r "s/([^=]*)=.*/\1/g")
        echo "$VAR_NAME=$(eval echo \$$VAR_NAME)"
        sed -i "s#{$VAR_NAME}#$(eval echo \$$VAR_NAME)#g" $CONFIG_DIR/conf.json
    fi
done

cd /opt/src
node index.js
