package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/constants"
)

// UpdatePodStatusWorkers is the count of goroutines of update pod status
var UpdateDeviceStatusWorkers int

func init() {
	if psw, err := config.CONFIG.GetValue("devicecontroller.update-pod-status-workers").ToInt(); err != nil {
		UpdateDeviceStatusWorkers = constants.DefaultUpdateDeviceStatusWorkers
	} else {
		UpdateDeviceStatusWorkers = psw
	}
	log.LOGGER.Infof("update device status workers: %d", UpdateDeviceStatusWorkers)
}
