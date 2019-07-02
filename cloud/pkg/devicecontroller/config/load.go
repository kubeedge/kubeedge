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
