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
	"encoding/json"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
	"github.com/kubeedge/kubeedge/tests/stubs/common/types"
	"github.com/kubeedge/kubeedge/tests/stubs/common/utils"
)

// NewUpstreamController creates a upstream controller
func NewUpstreamController(context *context.Context, pm *PodManager) (*UpstreamController, error) {
	// New upstream controller
	uc := &UpstreamController{context: context, podManager: pm}
	return uc, nil
}

// UpstreamController subscribe messages from edge
type UpstreamController struct {
	context             *context.Context
	podManager          *PodManager
	stopDispatch        chan struct{}
	stopUpdatePodStatus chan struct{}
	podStatusChan       chan model.Message
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	log.LOGGER.Infof("Start upstream controller")
	uc.stopDispatch = make(chan struct{})
	uc.stopUpdatePodStatus = make(chan struct{})
	uc.podStatusChan = make(chan model.Message, 1024)

	go uc.WaitforMessage(uc.stopDispatch)
	go uc.UpdatePodStatus(uc.stopUpdatePodStatus)

	return nil
}

// Stop UpstreamController
func (uc *UpstreamController) Stop() error {
	log.LOGGER.Infof("Stop upstream controller")
	uc.stopDispatch <- struct{}{}
	uc.stopUpdatePodStatus <- struct{}{}
	return nil
}

// WaitforMessage from cloudhub
func (uc *UpstreamController) WaitforMessage(stop chan struct{}) {
	running := true
	go func() {
		<-stop
		log.LOGGER.Infof("Stop waiting for message")
		running = false
	}()
	for running {
		// Receive message from cloudhub
		msg, err := uc.context.Receive(constants.ControllerStub)
		if err != nil {
			log.LOGGER.Errorf("Receive message failed: %v", err)
			continue
		}
		log.LOGGER.Debugf("Receive message: %v", msg)

		// Get resource type in message
		resourceType, err := utils.GetResourceType(msg)
		if err != nil {
			log.LOGGER.Errorf("Get message: %s resource type with error: %v", msg.GetID(), err)
			continue
		}
		log.LOGGER.Infof("Message: %s resource type: %s", msg.GetID(), resourceType)

		switch resourceType {
		case model.ResourceTypePodStatus:
			uc.podStatusChan <- msg
		default:
			log.LOGGER.Debugf("Message: %s, resource type: %s unsupported", msg.GetID(), resourceType)
		}
	}
}

// UpdatePodStatus is used to update pod status in cache map
func (uc *UpstreamController) UpdatePodStatus(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.podStatusChan:
			log.LOGGER.Infof("Message: %s operation: %s resource: %s",
				msg.GetID(), msg.GetOperation(), msg.GetResource())
			switch msg.GetOperation() {
			case model.UpdateOperation:
				// Marshal message content
				var data []byte
				switch msg.Content.(type) {
				case []byte:
					data = msg.GetContent().([]byte)
				default:
					var err error
					data, err = json.Marshal(msg.GetContent())
					if err != nil {
						log.LOGGER.Warnf("message: %s process failure, marshal content failed with error: %s", msg.GetID(), err)
						continue
					}
				}

				// Get pod
				var pod types.FakePod
				if err := json.Unmarshal(data, &pod); err != nil {
					log.LOGGER.Errorf("Unmarshal content failed with error: %s", msg.GetID(), err)
					continue
				}

				// Update pod status in cache
				uc.podManager.UpdatePodStatus(pod.Namespace+"/"+pod.Name, pod.Status)

				log.LOGGER.Infof("Pod namespace: %s name: %s status: %s",
					pod.Namespace, pod.Name, pod.Status)
			default:
				log.LOGGER.Debugf("Pod operation: %s unsupported", msg.GetOperation())
			}
			log.LOGGER.Debugf("Message: %s process successfully", msg.GetID())
		case <-stop:
			log.LOGGER.Infof("Stop updatePodStatus")
			running = false
		}
	}
}
