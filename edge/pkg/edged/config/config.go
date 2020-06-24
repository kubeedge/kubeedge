/*
Copyright 2019 The KubeEdge Authors.

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

package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const (
	//DockerEndpoint gives the default endpoint for docker engine
	DockerEndpoint = "unix:///var/run/docker.sock"

	//RemoteRuntimeEndpoint gives the default endpoint for CRI runtime
	RemoteRuntimeEndpoint = "unix:///var/run/dockershim.sock"

	//RemoteContainerRuntime give Remote container runtime name
	RemoteContainerRuntime = "remote"

	//MinimumEdgedMemoryCapacity gives the minimum default memory (2G) of edge
	MinimumEdgedMemoryCapacity = 2147483647

	//PodSandboxImage gives the default pause container image
	PodSandboxImage = "k8s.gcr.io/pause"

	// ImagePullProgressDeadlineDefault gives the default image pull progress deadline
	ImagePullProgressDeadlineDefault = 60
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.Edged
}

func InitConfigure(e *v1alpha1.Edged) {
	once.Do(func() {
		Config = Configure{
			Edged: *e,
		}
	})
}
