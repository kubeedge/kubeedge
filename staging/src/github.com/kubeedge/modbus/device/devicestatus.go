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
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/modbus/driver"
	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/grpcclient"
)

// DeviceStates is structure for getting device states.
type DeviceStates struct {
	Client          *driver.CustomizedClient
	DeviceName      string
	DeviceNamespace string
	ReportToCloud   bool
	ReportCycle     time.Duration
}

// Run timer function.
func (deviceStates *DeviceStates) PushStatesToEdgeCore() {
	states, err := deviceStates.Client.GetDeviceStates()
	if err != nil {
		klog.Errorf("GetDeviceStates failed: %v", err)
		return
	}

	statesRequest := &dmiapi.ReportDeviceStatesRequest{
		DeviceName:      deviceStates.DeviceName,
		State:           states,
		DeviceNamespace: deviceStates.DeviceNamespace,
	}

	klog.V(4).Infof("send device %s status %s request to cloud", statesRequest.DeviceName, statesRequest.State)
	if err = grpcclient.ReportDeviceStates(statesRequest); err != nil {
		klog.Errorf("fail to report device states of %s with err: %+v", deviceStates.DeviceName, err)
	}
}

func (deviceStates *DeviceStates) Run(ctx context.Context) {
	// No need to report device status to the cloud
	if !deviceStates.ReportToCloud {
		return
	}
	// Set device status report cycle
	if deviceStates.ReportCycle == 0 {
		deviceStates.ReportCycle = common.DefaultReportCycle
	}
	ticker := time.NewTicker(deviceStates.ReportCycle)
	for {
		select {
		case <-ticker.C:
			deviceStates.PushStatesToEdgeCore()
		case <-ctx.Done():
			return
		}
	}
}
