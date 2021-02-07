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
	"encoding/json"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common/util"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
	"github.com/kubeedge/kubeedge/tests/stubs/common/types"
)

// WaitforMessage is used to receive and process message
func (hs *HandlerStub) WaitforMessage() {
	go func() {
		for {
			select {
			case <-beehiveContext.Done():
				klog.Warning("stop waiting for message")
				return
			default:
			}
			if msg, err := beehiveContext.Receive(hs.Name()); err == nil {
				klog.V(4).Infof("Receive a message %v", msg)
				hs.ProcessMessage(msg)
			} else {
				klog.Errorf("Failed to receive message %v with error: %v", msg, err)
			}
		}
	}()
}

// ProcessMessage based on the operation type
func (hs *HandlerStub) ProcessMessage(msg model.Message) {
	klog.V(4).Infof("Begin to process message %v", msg)
	operation := msg.GetOperation()
	switch operation {
	case model.InsertOperation:
		hs.ProcessInsert(msg)
	case model.DeleteOperation:
		hs.ProcessDelete(msg)
	default:
		klog.V(4).Infof("Unsupported message: %s operation: %s", msg.GetID(), operation)
	}
	klog.V(4).Infof("End to process message %v", msg)
}

// ProcessInsert message
func (hs *HandlerStub) ProcessInsert(msg model.Message) {
	// Get resource type
	_, resType, _, err := util.ParseResourceEdge(msg.GetResource(), msg.GetOperation())
	if err != nil {
		klog.Errorf("failed to parse the Resource: %v", err)
		return
	}

	if resType == model.ResourceTypePod {
		// receive pod add event
		klog.V(4).Infof("Message content: %v", msg)

		// Marshal message content
		var data []byte
		switch msg.Content.(type) {
		case []byte:
			data = msg.GetContent().([]byte)
		default:
			data, err = json.Marshal(msg.GetContent())
			if err != nil {
				klog.Warningf("message: %s process failure, marshal content failed with error: %s", msg.GetID(), err)
				return
			}
		}

		// Get pod
		var pod types.FakePod
		if err := json.Unmarshal(data, &pod); err != nil {
			klog.Errorf("Unmarshal content failed with error: %s, %v", msg.GetID(), err)
			return
		}

		// Build Add message
		pod.Status = constants.PodRunning
		respMessage := model.NewMessage("")
		resource := pod.Namespace + "/" + model.ResourceTypePodStatus + "/" + pod.Name
		respMessage.Content = pod
		respMessage.BuildRouter(constants.HandlerStub, constants.GroupResource, resource, model.UpdateOperation)

		hs.SendToCloud(respMessage)

		// Add pod in cache
		hs.podManager.AddPod(pod.Namespace+"/"+pod.Name, pod)
	}
}

// ProcessDelete message
func (hs *HandlerStub) ProcessDelete(msg model.Message) {
	// Get resource type
	_, resType, _, err := util.ParseResourceEdge(msg.GetResource(), msg.GetOperation())
	if err != nil {
		klog.Errorf("failed to parse the Resource: %v", err)
		return
	}

	if resType == model.ResourceTypePod {
		// Receive pod delete event
		klog.V(4).Infof("Message content: %v", msg)

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
				return
			}
		}

		// Get pod
		var pod types.FakePod
		if err := json.Unmarshal(data, &pod); err != nil {
			klog.Errorf("Unmarshal content failed with error: %s, %v", msg.GetID(), err)
			return
		}
		// Delete pod in cache
		hs.podManager.DeletePod(pod.Namespace + "/" + pod.Name)
	}
}

// SendToCloud sends message to cloudhub by edgehub
func (hs *HandlerStub) SendToCloud(msg *model.Message) {
	klog.V(4).Infof("Begin to send message %v", *msg)
	beehiveContext.SendToGroup(constants.HubGroup, *msg)
	klog.V(4).Infof("End to send message %v", *msg)
}
