package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/constants"
)

// UpdateDeviceStatusBuffer is the size of channel which save update device status message from edge
var UpdateDeviceStatusBuffer int

func init() {
	if psb, err := config.CONFIG.GetValue("devicecontroller.update-device-status-buffer").ToInt(); err != nil {
		UpdateDeviceStatusBuffer = constants.DefaultUpdateDeviceStatusBuffer
	} else {
		UpdateDeviceStatusBuffer = psb
	}
	log.LOGGER.Infof("Update device status buffer: %d", UpdateDeviceStatusBuffer)
}
