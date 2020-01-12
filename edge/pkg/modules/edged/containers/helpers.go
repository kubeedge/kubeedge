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
	"time"

	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

//KubeSourcesReady is blank structure just for function referencing
type KubeSourcesReady struct{}

//AllReady give ready state of Kube Sources
func (s *KubeSourcesReady) AllReady() bool {
	return true
}

type containerRunner struct {
}

func (c *containerRunner) RunInContainer(id kubecontainer.ContainerID, cmd []string, timeout time.Duration) ([]byte, error) {
	return nil, nil
}

//NewContainerRunner returns container manager object
// TODO: we didn't realized Run In container interface yet
func NewContainerRunner() kubecontainer.ContainerCommandRunner {
	return &containerRunner{}
}
