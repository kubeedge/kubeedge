/*
Copyright 2022 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package image

import (
	"strings"
)

const (
	EdgePause = "pause"
	EdgeCore  = "edgecore"
	EdgeMQTT  = "mqtt"
)

type Set map[string]string

func (s Set) Current(ver string) Set {
	// To prevent the initial set from being modified,
	// a new set is returned anyway.
	set := make(Set)
	for k, v := range s {
		if ver == "" {
			set[k] = v
			continue
		}
		arr := strings.SplitN(v, ":", 2)
		set[k] = arr[0] + ":" + ver
	}
	return set
}

func (s Set) Get(name string) string {
	return s[name]
}

var edgeSet = Set{
	EdgeCore: "kubeedge/installation-package",
}

func EdgeSet(version string) Set {
	set := edgeSet.Current(version)
	set[EdgeMQTT] = "eclipse-mosquitto:1.6.15"
	set[EdgePause] = "kubeedge/pause:3.1"
	return set
}
