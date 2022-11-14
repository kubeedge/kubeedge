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

	"github.com/kubeedge/kubeedge/common/constants"
)

const (
	CloudAdmission         = "admission"
	CloudCloudcore         = "cloudcore"
	CloudIptablesManager   = "iptables-manager"
	CloudControllerManager = "controller-manager"
)

const (
	EdgePause = "pause"
	EdgeCore  = "edgecore"
	EdgeMQTT  = "mqtt"
)

type Set map[string]string

var cloudComponentSet = Set{
	CloudAdmission:         "kubeedge/admission",
	CloudCloudcore:         "kubeedge/cloudcore",
	CloudIptablesManager:   "kubeedge/iptables-manager",
	CloudControllerManager: "kubeedge/controller-manager",
}

var cloudThirdPartySet = Set{}

var edgeComponentSet = Set{
	EdgeCore: "kubeedge/installation-package",
}

var edgeThirdPartySet = Set{
	EdgeMQTT:  constants.DefaultMosquittoImage,
	EdgePause: constants.DefaultPodSandboxImage,
}

func EdgeSet(imageRepository, version string) Set {
	set := edgeComponentSet.Current(imageRepository, version)
	thirdSet := edgeThirdPartySet.Current(imageRepository, "")
	set = set.Merge(thirdSet)
	return set
}

func CloudSet(imageRepository, version string) Set {
	set := cloudComponentSet.Current(imageRepository, version)
	thirdSet := cloudThirdPartySet.Current(imageRepository, "")
	set = set.Merge(thirdSet)
	return set
}

// Current replace repository and version for set
func (s Set) Current(imageRepository, ver string) Set {
	// To prevent the initial set from being modified,
	// a new set is returned anyway.
	set := make(Set)

	for k, v := range s {
		cur := v
		if ver != "" {
			arr := strings.SplitN(v, ":", 2)
			cur = arr[0] + ":" + ver
		}
		if imageRepository != "" {
			arr := strings.SplitN(cur, "/", 2)
			cur = imageRepository + "/" + arr[0]
			if len(arr) == 2 {
				cur = imageRepository + "/" + arr[1]
			}
		}
		set[k] = cur
	}

	return set
}

func (s Set) Get(name string) string {
	return s[name]
}

func (s Set) Merge(src Set) Set {
	for k, v := range src {
		s[k] = v
	}
	return s
}

func (s Set) List() []string {
	var result []string
	for _, v := range s {
		result = append(result, v)
	}
	return result
}
