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
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// Init module
func init() {
	core.Register(&HandlerStub{})
}

// HandlerStub definition
type HandlerStub struct {
	context    *context.Context
	podManager *PodManager
	stopChan   chan bool
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
func (hs *HandlerStub) Start(c *context.Context) {
	hs.context = c
	hs.stopChan = make(chan bool)

	// New pod manager
	pm, err := NewPodManager()
	if err != nil {
		log.LOGGER.Errorf("Failed to create pod manager with error: %v", err)
		return
	}
	hs.podManager = pm

	// Wait for message
	log.LOGGER.Infof("Wait for message")
	hs.WaitforMessage()

	// Start upstream controller
	upstream, err := NewUpstreamController(hs.context, pm)
	if err != nil {
		log.LOGGER.Errorf("New upstream controller failed with error: %v", err)
		return
	}
	upstream.Start()

	// Receive stop signal
	<-hs.stopChan
	upstream.Stop()
}

// Cleanup resources
func (hs *HandlerStub) Cleanup() {
	hs.stopChan <- true
	hs.context.Cleanup(hs.Name())
}
