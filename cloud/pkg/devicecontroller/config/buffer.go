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

// Package config
package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

// UpdateDeviceStatusBuffer is the size of channel which save update device status message from edge
var UpdateDeviceStatusBuffer int

// DeviceEventBuffer is the size of channel which save device event from k8s
var DeviceEventBuffer int

// DeviceModelEventBuffer is the size of channel which save devicemodel event from k8s
var DeviceModelEventBuffer int

func init() {
	if psb, err := config.CONFIG.GetValue("devicecontroller.buffer.update-device-status").ToInt(); err != nil {
		UpdateDeviceStatusBuffer = constants.DefaultUpdateDeviceStatusBuffer
	} else {
		UpdateDeviceStatusBuffer = psb
	}
	log.LOGGER.Infof("Update devicecontroller.buffer.update-device-status: %d", UpdateDeviceStatusBuffer)

	if deb, err := config.CONFIG.GetValue("devicecontroller.buffer.device-event").ToInt(); err != nil {
		DeviceEventBuffer = constants.DefaultDeviceEventBuffer
	} else {
		DeviceEventBuffer = deb
	}
	log.LOGGER.Infof("Update devicecontroller.buffer.device-event: %d", DeviceEventBuffer)

	if dmeb, err := config.CONFIG.GetValue("devicecontroller.buffer.device-model-event").ToInt(); err != nil {
		DeviceModelEventBuffer = constants.DefaultDeviceModelEventBuffer
	} else {
		DeviceModelEventBuffer = dmeb
	}
	log.LOGGER.Infof("Update devicecontroller.buffer.device-model-event: %d", DeviceModelEventBuffer)

}
