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
	"encoding/json"

	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util/controller"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

// UpstreamController subscribe messages from edge and sync to k8s api server
type UpstreamController struct {
	// downstream controller to update NodeUpgradeJob status in cache
	dc *DownstreamController

	kubeClient   kubernetes.Interface
	informer     k8sinformer.SharedInformerFactory
	crdClient    crdClientset.Interface
	messageLayer messagelayer.MessageLayer
	// message channel
	taskStatusChan chan model.Message
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("Start Task Upstream Controller")

	uc.taskStatusChan = make(chan model.Message, config.Config.Buffer.TaskStatus)
	go uc.dispatchMessage()

	for i := 0; i < int(config.Config.Load.TaskWorkers); i++ {
		go uc.updateTaskStatus()
	}
	return nil
}

// Start UpstreamController
func (uc *UpstreamController) dispatchMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop dispatch task upstream message")
			return
		default:
		}

		msg, err := uc.messageLayer.Receive()
		if err != nil {
			klog.Warningf("Receive message failed, %v", err)
			continue
		}

		klog.V(4).Infof("task upstream controller receive msg %#v", msg)

		uc.taskStatusChan <- msg
	}
}

// updateTaskStatus update NodeUpgradeJob status field
func (uc *UpstreamController) updateTaskStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop update NodeUpgradeJob status")
			return
		case msg := <-uc.taskStatusChan:
			klog.V(4).Infof("Message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			// get nodeID and upgradeID from Upgrade msg:
			nodeID := util.GetNodeName(msg.GetResource())
			taskID := util.GetTaskID(msg.GetResource())

			data, err := msg.GetContentData()
			if err != nil {
				klog.Errorf("failed to get node upgrade content data: %v", err)
				continue
			}

			c, err := controller.GetController(msg.GetOperation())
			if err != nil {
				klog.Errorf("Failed to get controller: %v", err)
				continue
			}

			resp := types.NodeTaskResponse{}
			err = json.Unmarshal(data, &resp)
			if err != nil {
				klog.Errorf("Failed to unmarshal node upgrade response: %v", err)
				continue
			}
			event := fsm.Event{
				Type:            resp.Event,
				Action:          resp.Action,
				Msg:             resp.Reason,
				ExternalMessage: resp.ExternalMessage,
			}

			_, err = c.ReportNodeStatus(taskID, nodeID, event)
			if err != nil {
				klog.Errorf("Failed to report status: %v", err)
				continue
			}
		}
	}
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController(dc *DownstreamController) (*UpstreamController, error) {
	uc := &UpstreamController{
		kubeClient:   keclient.GetKubeClient(),
		informer:     informers.GetInformersManager().GetKubeInformerFactory(),
		crdClient:    keclient.GetCRDClient(),
		messageLayer: messagelayer.TaskManagerMessageLayer(),
		dc:           dc,
	}
	return uc, nil
}
