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

kill_edge_core() {
   sudo pkill edge_core
    #kill the edge_core process if it exists.
    sleep 5s
    if pgrep edge_core >/dev/null
    then
        echo "Failed to kill edge_core process !!"
        exit 1
    else
        echo "edge_core is successfully killed !!"
    fi
}

kill_edgecontroller() {
    sudo pkill edge
    #kill the edgecontroller process if it exists.
    sleep 5s
    if pgrep edgecontroller >/dev/null
    then
        echo "Failed to kill the edgecontroller !!"
        exit 1
    else
        echo "edgecontroller is successfully killed !!"
    fi
}

kill_edge_core
kill_edgecontroller
