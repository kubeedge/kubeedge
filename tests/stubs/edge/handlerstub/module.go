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

package handlerstub

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// Init module
func init() {
	core.Register(&HandlerStub{})
}

// HandlerStub definition
type HandlerStub struct {
	podManager *PodManager
}

func (*HandlerStub) Enable() bool {
	return true
}

// Return module name
func (*HandlerStub) Name() string {
	return constants.HandlerStub
}

// Return module group
func (*HandlerStub) Group() string {
	return constants.MetaGroup
}

// Start handler hub
func (hs *HandlerStub) Start() {
	// New pod manager
	pm, err := NewPodManager()
	if err != nil {
		klog.Errorf("Failed to create pod manager with error: %v", err)
		return
	}
	hs.podManager = pm

	// Wait for message
	klog.Infof("Wait for message")
	hs.WaitforMessage()

	// Start upstream controller
	upstream, err := NewUpstreamController(pm)
	if err != nil {
		klog.Errorf("New upstream controller failed with error: %v", err)
		return
	}
	if err := upstream.Start(); err != nil {
		klog.Errorf("Failed to start upstream with error: %v", err)
		return
	}
}
