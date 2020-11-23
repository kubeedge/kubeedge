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

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
	"github.com/kubeedge/kubeedge/tests/stubs/common/types"
	"github.com/kubeedge/kubeedge/tests/stubs/common/utils"
)

// NewUpstreamController creates a upstream controller
func NewUpstreamController(pm *PodManager) (*UpstreamController, error) {
	// New upstream controller
	uc := &UpstreamController{podManager: pm}
	return uc, nil
}

// UpstreamController subscribe messages from edge
type UpstreamController struct {
	podManager    *PodManager
	podStatusChan chan model.Message
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Infof("Start upstream controller")
	uc.podStatusChan = make(chan model.Message, 1024)

	go uc.WaitforMessage()
	go uc.UpdatePodStatus()

	return nil
}

// WaitforMessage from cloudhub
func (uc *UpstreamController) WaitforMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Infof("Stop waiting for message")
			return
		default:
		}
		// Receive message from cloudhub
		msg, err := beehiveContext.Receive(constants.ControllerStub)
		if err != nil {
			klog.Errorf("Receive message failed: %v", err)
			continue
		}
		klog.V(4).Infof("Receive message: %v", msg)

		// Get resource type in message
		resourceType, err := utils.GetResourceType(msg)
		if err != nil {
			klog.Errorf("Get message: %s resource type with error: %v", msg.GetID(), err)
			continue
		}
		klog.Infof("Message: %s resource type: %s", msg.GetID(), resourceType)

		switch resourceType {
		case model.ResourceTypePodStatus:
			uc.podStatusChan <- msg
		default:
			klog.V(4).Infof("Message: %s, resource type: %s unsupported", msg.GetID(), resourceType)
		}
	}
}

// UpdatePodStatus is used to update pod status in cache map
func (uc *UpstreamController) UpdatePodStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Infof("Stop updatePodStatus")
			return
		case msg := <-uc.podStatusChan:
			klog.Infof("Message: %s operation: %s resource: %s",
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
						klog.Warningf("message: %s process failure, marshal content failed with error: %s", msg.GetID(), err)
						continue
					}
				}

				// Get pod
				var pod types.FakePod
				if err := json.Unmarshal(data, &pod); err != nil {
					klog.Errorf("Unmarshal content failed with error: %s, %v", msg.GetID(), err)
					continue
				}

				// Update pod status in cache
				uc.podManager.UpdatePodStatus(pod.Namespace+"/"+pod.Name, pod.Status)

				klog.Infof("Pod namespace: %s name: %s status: %s",
					pod.Namespace, pod.Name, pod.Status)
			default:
				klog.V(4).Infof("Pod operation: %s unsupported", msg.GetOperation())
			}
			klog.V(4).Infof("Message: %s process successfully", msg.GetID())
		}
	}
}
