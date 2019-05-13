package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

// UpdateDeviceStatusWorkers is the count of goroutines of update device status
var UpdateDeviceStatusWorkers int

func init() {
	if psw, err := config.CONFIG.GetValue("devicecontroller.load.update-device-status-workers").ToInt(); err != nil {
		UpdateDeviceStatusWorkers = constants.DefaultUpdateDeviceStatusWorkers
	} else {
		UpdateDeviceStatusWorkers = psw
	}
	log.LOGGER.Infof("Update device status workers: %d", UpdateDeviceStatusWorkers)
}
