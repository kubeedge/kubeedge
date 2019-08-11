#!/bin/bash

###
#Copyright 2019 The KubeEdge Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
###


function checkModified {
	modified=$( git status --short 2>/dev/null | grep -e "^.M" | wc -l)

	if [ ${modified} -eq 0 ]; then
		echo 0
		return
	fi

	echo 2
}
 
go mod vendor
ret=$(checkModified)
if [ ${ret} -eq 0 ]; then
	echo "SUCCESS: vendor is up to date"
else
	echo  "FAILED: vendor needs an update; The diff is:"
	git diff
	exit 1
fi

go mod tidy
ret=$(checkModified)
if [ ${ret} -eq 0 ]; then
	echo "SUCCESS: go.mod and go.sum are in tiny"
else
	echo  "FAILED: go.mod / go.dum needs an update; The diff is:"
	git diff
	exit 1
fi
