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

package device

import (
	"k8s.io/klog/v2"

	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus-go/driver"
	"github.com/kubeedge/kubeedge/mappers/modbus-go/globals"
)

// GetStatus is the timer structure for getting device status.
type GetStatus struct {
	Client *driver.ModbusClient
	Status string
	topic  string
}

// Run timer function.
func (gs *GetStatus) Run() {
	gs.Status = gs.Client.GetStatus()

	var payload []byte
	var err error
	if payload, err = mappercommon.CreateMessageState(gs.Status); err != nil {
		klog.Error("Create message state failed: ", err)
		return
	}
	if err = globals.MqttClient.Publish(gs.topic, payload); err != nil {
		klog.Error("Publish failed: ", err)
		return
	}
}
