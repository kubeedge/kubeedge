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

setuptype=$1

kill_component() {
    local component=$1
    if pgrep "$component" &>/dev/null; then
        # edgesite process is found, kill the process.
        sudo pkill $component &>/dev/null
        if [[ "$?" == "0" ]]; then
            echo "$component is successfully killed !!"
        else
            echo "Failed to kill $component process !!"
            exit 1
        fi
    fi
}

kill_all_components() {
    local components="cloudcore edgecore edgesite"
    for component in $components; do
        kill_component "$component"
    done
}

cleanup_files(){
    sudo rm -rf /etc/kubeedge /var/lib/kubeedge
    sudo rm -f tests/e2e/config.json
    find -name "*.test" | xargs sudo rm -f
}

kill_all_components

cleanup_files
