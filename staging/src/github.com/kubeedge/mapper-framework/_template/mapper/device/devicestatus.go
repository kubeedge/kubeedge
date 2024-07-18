/*
Copyright 2024 The KubeEdge Authors.

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

package device

import (
	"context"
	"log"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/driver"
	dmiapi "github.com/kubeedge/kubeedge/pkg/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/grpcclient"
)

// DeviceStates is structure for getting device states.
type DeviceStates struct {
	Client          *driver.CustomizedClient
	DeviceName      string
	DeviceNamespace string
}

// Run timer function.
func (deviceStates *DeviceStates) PushStatesToEdgeCore() {
	states, error := deviceStates.Client.GetDeviceStates()
	if error != nil {
		klog.Errorf("GetDeviceStates failed: %v", error)
		return
	}

	statesRequest := &dmiapi.ReportDeviceStatesRequest{
		DeviceName:      deviceStates.DeviceName,
		State:           states,
		DeviceNamespace: deviceStates.DeviceNamespace,
	}

	log.Printf("send statesRequest", statesRequest.DeviceName, statesRequest.State)
	if err := grpcclient.ReportDeviceStates(statesRequest); err != nil {
		klog.Errorf("fail to report device states of %s with err: %+v", deviceStates.DeviceName, err)
	}
}

func (deviceStates *DeviceStates) Run(ctx context.Context) {
	// TODO setting states reportCycle
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			deviceStates.PushStatesToEdgeCore()
		case <-ctx.Done():
			return
		}
	}
}
