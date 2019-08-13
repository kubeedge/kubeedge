package config

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
)

// UpdateDeviceStatusWorkers is the count of goroutines of update device status
var UpdateDeviceStatusWorkers int

func InitLoadConfig() {
	if psw, err := config.CONFIG.GetValue("devicecontroller.load.update-device-status-workers").ToInt(); err != nil {
		UpdateDeviceStatusWorkers = constants.DefaultUpdateDeviceStatusWorkers
	} else {
		UpdateDeviceStatusWorkers = psw
	}
	klog.Infof("Update device status workers: %d", UpdateDeviceStatusWorkers)
}
