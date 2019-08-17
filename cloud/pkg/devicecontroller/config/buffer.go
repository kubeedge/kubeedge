package config

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

// UpdateDeviceStatusBuffer is the size of channel which save update device status message from edge
var UpdateDeviceStatusBuffer int

// DeviceEventBuffer is the size of channel which save device event from k8s
var DeviceEventBuffer int

// DeviceModelEventBuffer is the size of channel which save devicemodel event from k8s
var DeviceModelEventBuffer int

func InitBufferConfig() {
	if psb, err := config.CONFIG.GetValue("devicecontroller.buffer.update-device-status").ToInt(); err != nil {
		UpdateDeviceStatusBuffer = constants.DefaultUpdateDeviceStatusBuffer
	} else {
		UpdateDeviceStatusBuffer = psb
	}
	klog.Infof("Update devicecontroller.buffer.update-device-status: %d", UpdateDeviceStatusBuffer)

	if deb, err := config.CONFIG.GetValue("devicecontroller.buffer.device-event").ToInt(); err != nil {
		DeviceEventBuffer = constants.DefaultDeviceEventBuffer
	} else {
		DeviceEventBuffer = deb
	}
	klog.Infof("Update devicecontroller.buffer.device-event: %d", DeviceEventBuffer)

	if dmeb, err := config.CONFIG.GetValue("devicecontroller.buffer.device-model-event").ToInt(); err != nil {
		DeviceModelEventBuffer = constants.DefaultDeviceModelEventBuffer
	} else {
		DeviceModelEventBuffer = dmeb
	}
	klog.Infof("Update devicecontroller.buffer.device-model-event: %d", DeviceModelEventBuffer)

}
