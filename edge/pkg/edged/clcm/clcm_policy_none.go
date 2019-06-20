/*
Copyright 2016 The Kubernetes Authors.

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
This file is derived from K8S Kubelet code with reduced set of methods
*/

package clcm

import (
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager"
)

const (
	//PolicyNone define no policy
	PolicyNone      = "none"
	reconcilePeriod = 1 * time.Second
)

type containerLifecycleManagerImpl struct {
	cpuManager cpumanager.Manager
}

var _ ContainerLifecycleManager = &containerLifecycleManagerImpl{}

// NewContainerLifecycleManager create new object for container lifecycle manager
func NewContainerLifecycleManager(kubeletRootDir string) (ContainerLifecycleManager, error) {
	var err error
	clcm := &containerLifecycleManagerImpl{}
	result := make(v1.ResourceList)
	clcm.cpuManager, err = cpumanager.NewManager(
		PolicyNone,
		reconcilePeriod,
		nil,
		result,
		kubeletRootDir,
	)
	if err != nil {
		glog.Errorf("failed to initialize cpu manager: %v", err)
		return nil, err
	}
	return clcm, nil
}

func (clcm *containerLifecycleManagerImpl) InternalContainerLifecycle() InternalContainerLifecycle {
	return &internalContainerLifecycleImpl{clcm.cpuManager}
}
