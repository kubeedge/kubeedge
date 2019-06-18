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
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// NewDownstreamController creates a downstream controller
func NewDownstreamController(context *context.Context, pm *PodManager) (*DownstreamController, error) {
	// New downstream controller
	dc := &DownstreamController{context: context, podManager: pm}
	return dc, nil
}

// DownstreamController receives http request and send to cloudhub
type DownstreamController struct {
	context    *context.Context
	podManager *PodManager
	podStop    chan struct{}
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	log.LOGGER.Infof("Start downstream controller")
	dc.podStop = make(chan struct{})
	go dc.SyncPods(dc.podStop)
	return nil
}

// Stop DownstreamController
func (dc *DownstreamController) Stop() error {
	log.LOGGER.Infof("Stop downstream controller")
	dc.podStop <- struct{}{}
	return nil
}

// SyncPods is used to send message to cloudhub
func (dc *DownstreamController) SyncPods(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-dc.podManager.GetEvent():
			log.LOGGER.Infof("Send message to cloudhub: %v", *msg)
			dc.context.Send(constants.CloudHub, *msg)
			log.LOGGER.Infof("Finish send message to cloudhub")
		case <-stop:
			log.LOGGER.Infof("Stop sync pod")
			running = false
		}
	}
}
