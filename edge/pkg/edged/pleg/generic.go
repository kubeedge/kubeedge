/*
Copyright 2017 The Kubernetes Authors.

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

package pleg

import (
	"time"

	"k8s.io/apimachinery/pkg/util/clock"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/pleg"
)

//GenericLifecycle is object for pleg lifecycle
type GenericLifecycle struct {
	pleg.GenericPLEG
}

//NewGenericLifecycleRemote creates new generic life cycle object for remote
func NewGenericLifecycleRemote(runtime kubecontainer.Runtime, channelCapacity int,
	relistPeriod time.Duration, podCache kubecontainer.Cache, clock clock.Clock) pleg.PodLifecycleEventGenerator {
	//kubeContainerManager := containers.NewKubeContainerRuntime(manager)
	genericPLEG := pleg.NewGenericPLEG(runtime, channelCapacity, relistPeriod, podCache, clock)
	return &GenericLifecycle{
		GenericPLEG: *genericPLEG.(*pleg.GenericPLEG),
	}
}

// Start spawns a goroutine to relist periodically.
func (gl *GenericLifecycle) Start() {
	gl.GenericPLEG.Start()
}
