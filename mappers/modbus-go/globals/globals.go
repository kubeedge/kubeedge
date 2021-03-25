/*
Copyright 2020 The KubeEdge Authors.

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

package globals

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus-go/driver"
)

// ModbusDev is the modbus device configuration and client information.
type ModbusDev struct {
	Instance v1alpha2.Device
	// Instance     mappercommon.DeviceInstance
	ModbusClient *driver.ModbusClient
}

var MqttClient mappercommon.MqttClient
