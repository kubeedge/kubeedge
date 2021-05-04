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
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
)

// NewDownstreamController creates a downstream controller
func NewDownstreamController(pm *PodManager) (*DownstreamController, error) {
	// New downstream controller
	dc := &DownstreamController{podManager: pm}
	return dc, nil
}

// DownstreamController receives http request and send to cloudhub
type DownstreamController struct {
	podManager *PodManager
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Infof("Start downstream controller")
	go dc.SyncPods()
	return nil
}

// SyncPods is used to send message to cloudhub
func (dc *DownstreamController) SyncPods() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop sync pod")
			return
		case msg := <-dc.podManager.GetEvent():
			klog.Infof("Send message to cloudhub: %v", *msg)
			beehiveContext.Send(constants.CloudHub, *msg)
			klog.Info("Finish send message to cloudhub")
		}
	}
}
