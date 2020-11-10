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

package controllerstub

import (
	"net/http"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// Init module
func init() {
	core.Register(&ControllerStub{})
}

// HandlerStub definition
type ControllerStub struct {
}

func (*ControllerStub) Enable() bool {
	return true
}

// Return module name
func (*ControllerStub) Name() string {
	return constants.ControllerStub
}

// Return module group
func (*ControllerStub) Group() string {
	return constants.ControllerGroup
}

// Start controller hub
func (cs *ControllerStub) Start() {
	// New pod manager
	pm, err := NewPodManager()
	if err != nil {
		klog.Errorf("Failed to create pod manager with error: %v", err)
		return
	}

	// Start downstream controller
	downstream, err := NewDownstreamController(pm)
	if err != nil {
		klog.Errorf("New downstream controller failed with error: %v", err)
		return
	}
	if err := downstream.Start(); err != nil {
		klog.Errorf("Start downstream controller failed with error: %v", err)
		return
	}

	// Start upstream controller
	upstream, err := NewUpstreamController(pm)
	if err != nil {
		klog.Errorf("New upstream controller failed with error: %v", err)
		return
	}
	if err := upstream.Start(); err != nil {
		klog.Errorf("Start upstream controller failed with error: %v", err)
		return
	}

	// Start http server
	http.HandleFunc(constants.PodResource, pm.PodHandlerFunc)
	klog.Info("Start http service")
	go func() {
		if err := http.ListenAndServe(":54321", nil); err != nil {
			klog.Errorf("Start http service failed with error: %v", err)
		}
	}()
}
