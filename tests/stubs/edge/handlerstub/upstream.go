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

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// NewUpstreamController creates a upstream controller
func NewUpstreamController(pm *PodManager) (*UpstreamController, error) {
	// New upstream controller
	dc := &UpstreamController{podManager: pm}
	return dc, nil
}

// UpstreamController sends message to edghub
type UpstreamController struct {
	podManager *PodManager
}

// Start upstream
func (dc *UpstreamController) Start() error {
	klog.Infof("Start upstream controller")
	go dc.SyncPods()
	return nil
}

// SyncPods is used to send simulation messages to edgehub periodically
func (dc *UpstreamController) SyncPods() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Infof("Stop sync pods")
			return
		default:
		}
		pods := dc.podManager.ListPods()
		klog.V(4).Infof("Current pods number is: %v", len(pods))
		for _, pod := range pods {
			// Periodic sync message
			msg := model.NewMessage("")
			resource := pod.Namespace + "/" + model.ResourceTypePodStatus + "/" + pod.Name
			msg.Content = pod
			msg.BuildRouter(constants.HandlerStub, constants.GroupResource, resource, model.UpdateOperation)

			klog.V(4).Infof("Begin to sync message: %v", *msg)
			beehiveContext.SendToGroup(constants.HubGroup, *msg)
			klog.V(4).Infof("End to sync message: %v", *msg)
		}
		time.Sleep(5 * time.Second)
	}
}
