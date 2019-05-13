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
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// NewUpstreamController creates a upstream controller
func NewUpstreamController(context *context.Context, pm *PodManager) (*UpstreamController, error) {
	// New upstream controller
	dc := &UpstreamController{context: context, podManager: pm}
	return dc, nil
}

// UpstreamController sends message to edghub
type UpstreamController struct {
	context    *context.Context
	podManager *PodManager
	podStop    chan struct{}
}

// Start upstream
func (dc *UpstreamController) Start() error {
	log.LOGGER.Infof("Start upstream controller")
	dc.podStop = make(chan struct{})
	go dc.SyncPods(dc.podStop)
	return nil
}

// Stop UpstreamController
func (dc *UpstreamController) Stop() error {
	log.LOGGER.Infof("Stop upstream controller")
	dc.podStop <- struct{}{}
	return nil
}

// SyncPods is used to send simulation messages to edgehub periodically
func (dc *UpstreamController) SyncPods(stop chan struct{}) {
	running := true
	go func() {
		<-stop
		log.LOGGER.Infof("Stop sync pods")
		running = false
	}()
	for running {
		pods := dc.podManager.ListPods()
		log.LOGGER.Debugf("Current pods number is: %v", len(pods))
		for _, pod := range pods {
			// Periodic sync message
			msg := model.NewMessage("")
			resource := pod.Namespace + "/" + model.ResourceTypePodStatus + "/" + pod.Name
			msg.Content = pod
			msg.BuildRouter(constants.HandlerStub, constants.GroupResource, resource, model.UpdateOperation)

			log.LOGGER.Debugf("Begin to sync message: %v", *msg)
			dc.context.Send2Group(constants.HubGroup, *msg)
			log.LOGGER.Debugf("End to sync message: %v", *msg)
		}
		time.Sleep(5 * time.Second)
	}
}
