/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with pruned structures and interfaces
and changed most of the realization.
Changes done are
1. Some helper functions are derived from k8s.io\kubernetes\pkg\kubelet\dockershim\helpers.go
*/

package containers

import (
	"fmt"
	"strings"

	"github.com/kubeedge/beehive/pkg/common/log"
	"k8s.io/api/core/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/container"
)

type sourceImpl struct{}

func (s sourceImpl) AddSource(source string) {
}

func (s sourceImpl) AllReady() bool {
	return true
}

//ConvertEnvVersion converts environment version
// TODO: need consider EnvVar.ValueFrom
func ConvertEnvVersion(envs []v1.EnvVar) []container.EnvVar {
	var res []container.EnvVar
	for _, env := range envs {
		res = append(res, container.EnvVar{Name: env.Name, Value: env.Value})
	}
	return res
}

//GenerateEnvList generates environments list
func GenerateEnvList(envs []v1.EnvVar) (result []string) {
	for _, env := range envs {
		result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return
}

//EnableHostUserNamespace checks security to enable host user namespace
func EnableHostUserNamespace(pod *v1.Pod) bool {
	if pod.Spec.Containers[0].SecurityContext != nil && pod.Spec.Containers[0].SecurityContext.Privileged != nil && *pod.Spec.Containers[0].SecurityContext.Privileged {
		return true
	}
	return false
}

// GenerateMountBindings converts the mount list to a list of strings that
// can be understood by docker.
// '<HostPath>:<ContainerPath>[:options]', where 'options'
// is a comma-separated list of the following strings:
// 'ro', if the path is read only
// 'Z', if the volume requires SELinux relabeling
// propagation mode such as 'rslave'
func GenerateMountBindings(mounts []*container.Mount) []string {
	result := make([]string, 0, len(mounts))
	for _, m := range mounts {
		bind := fmt.Sprintf("%s:%s", m.HostPath, m.ContainerPath)
		var attrs []string
		if m.ReadOnly {
			attrs = append(attrs, "ro")
		}
		// Only request relabeling if the pod provides an SELinux context. If the pod
		// does not provide an SELinux context relabeling will label the volume with
		// the container's randomly allocated MCS label. This would restrict access
		// to the volume to the container which mounts it first.
		if m.SELinuxRelabel {
			attrs = append(attrs, "Z")
		}
		switch m.Propagation {
		case runtimeapi.MountPropagation_PROPAGATION_PRIVATE:
			// noop, private is default
		case runtimeapi.MountPropagation_PROPAGATION_BIDIRECTIONAL:
			attrs = append(attrs, "rshared")
		case runtimeapi.MountPropagation_PROPAGATION_HOST_TO_CONTAINER:
			attrs = append(attrs, "rslave")
		default:
			log.LOGGER.Warnf("unknown propagation mode for hostPath %q", m.HostPath)
			// Falls back to "private"
		}

		if len(attrs) > 0 {
			bind = fmt.Sprintf("%s:%s", bind, strings.Join(attrs, ","))
		}
		result = append(result, bind)
	}
	return result
}

//NewKubeContainerRuntime returns runtime object of container manager
func NewKubeContainerRuntime(cm ContainerManager) container.Runtime {
	return cm.(*containerManager)
}

//KubeSourcesReady is blank structure just for function referencing
type KubeSourcesReady struct{}

//AllReady give ready state of Kube Sources
func (s *KubeSourcesReady) AllReady() bool {
	return true
}

//NewContainerRunner returns container manager object
// TODO: we didn't realized Run In container interface yet
func NewContainerRunner() container.ContainerCommandRunner {
	return &containerManager{}
}
