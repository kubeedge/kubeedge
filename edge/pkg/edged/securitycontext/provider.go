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

package securitycontext

import (
	"strconv"

	"github.com/docker/docker/api/types/container"
	"k8s.io/api/core/v1"
)

//SimpleSecurityContextProvider is object ot define security context provider
type SimpleSecurityContextProvider struct{}

//NewSimpleSecurityContextProvider initialises and returns security context provider
func NewSimpleSecurityContextProvider() Provider {
	return SimpleSecurityContextProvider{}
}

//ModifyContainerConfig changes security context of container
func (s SimpleSecurityContextProvider) ModifyContainerConfig(pod *v1.Pod, config *container.Config) {
	securityContext := pod.Spec.Containers[0].SecurityContext
	if securityContext == nil {
		return
	}
	if securityContext.RunAsUser != nil {
		config.User = strconv.Itoa(int(*securityContext.RunAsUser))
	}
}

//ModifyHostConfig modifies security context of host
func (s SimpleSecurityContextProvider) ModifyHostConfig(pod *v1.Pod, hostConfig *container.HostConfig) {
	securityContext := pod.Spec.Containers[0].SecurityContext
	if securityContext != nil && securityContext.Privileged != nil {
		hostConfig.Privileged = *securityContext.Privileged
	}
}
