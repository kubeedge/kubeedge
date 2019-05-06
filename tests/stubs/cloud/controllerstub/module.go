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

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// Init module
func init() {
	core.Register(&ControllerStub{})
}

// HandlerStub definition
type ControllerStub struct {
	context  *context.Context
	stopChan chan bool
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
func (cs *ControllerStub) Start(c *context.Context) {
	cs.context = c
	cs.stopChan = make(chan bool)

	// New pod manager
	pm, err := NewPodManager()
	if err != nil {
		log.LOGGER.Errorf("Failed to create pod manager with error: %v", err)
		return
	}

	// Start downstream controller
	downstream, err := NewDownstreamController(cs.context, pm)
	if err != nil {
		log.LOGGER.Errorf("New downstream controller failed with error: %v", err)
		return
	}
	downstream.Start()

	// Start upstream controller
	upstream, err := NewUpstreamController(cs.context, pm)
	if err != nil {
		log.LOGGER.Errorf("New upstream controller failed with error: %v", err)
		return
	}
	upstream.Start()

	// Start http server
	http.HandleFunc(constants.PodResource, pm.PodHandlerFunc)
	go http.ListenAndServe(":54321", nil)
	log.LOGGER.Info("Start http service")

	// Receive stop signal
	<-cs.stopChan
	upstream.Stop()
	downstream.Stop()
}

// Cleanup resources
func (cs *ControllerStub) Cleanup() {
	cs.stopChan <- true
	cs.context.Cleanup(cs.Name())
}
