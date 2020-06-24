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

cd $GOPATH/src/github.com/kubeedge/kubeedge/edge

#removes bin folder which is created after running make verify
remove_folder() {
if [ -d bin ]; then
rm -rf bin
fi
}

#terminate edge core if running
stop_edgecore () {
if pgrep edgecore >/dev/null 2>&1 ; then
     pkill -9 edgecore
fi
}

#delete logs,covergage and database related files
cleanup_files () {
if [ -f edgecore ]; then
rm -f edgecore
fi

find . -type f -name "*db" -exec rm -f {} \;
find . -type f -name "*log" -exec rm -f {} \;
find . -type f -name "*out" -exec rm -f {} \;
}

remove_folder
stop_edgecore
cleanup_files
