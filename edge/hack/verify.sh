#!/usr/bin/env bash

# Copyright 2018 The KubeEdge Authors.
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

# get gometalinter(https://github.com/alecthomas/gometalinter)

curl -L https://git.io/vp6lP | sh
export PATH=${PATH}:${GOPATH}/bin:${GOPATH}/src/github.com/kubeedge/kubeedge/edge/bin

gometalinter --disable-all --enable=gofmt --enable=misspell --enable=golint --exclude=vendor --exclude=test ./...
if [ $? != 0 ]; then
        echo "Please fix the warnings!"
	echo "Run hack/update-gofmt.sh if any warnings in gofmt"
        exit 1
else
echo "Gofmt,misspell,golint checks have been passed"
fi
