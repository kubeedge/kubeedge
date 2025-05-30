/*
Copyright 2023 The KubeEdge Authors.

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

package manager

import (
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
)

type DownstreamController struct {
	downStreamChan chan model.Message
	messageLayer   messagelayer.MessageLayer
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("Start TaskManager Downstream Controller")

	go dc.syncTask()

	return nil
}

// syncTask is used to get events from informer
func (dc *DownstreamController) syncTask() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync tasks")
			return
		case msg := <-dc.downStreamChan:
			err := dc.messageLayer.Send(msg)
			if err != nil {
				klog.Errorf("Failed to send upgrade message %v due to error %v", msg.GetID(), err)
				return
			}
		}
	}
}

func NewDownstreamController(messageChan chan model.Message) (*DownstreamController, error) {
	dc := &DownstreamController{
		downStreamChan: messageChan,
		messageLayer:   messagelayer.TaskManagerMessageLayer(),
	}
	return dc, nil
}
