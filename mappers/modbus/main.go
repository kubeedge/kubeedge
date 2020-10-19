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

package main

import (
	"fmt"
	"os"

	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus/dev"
	"github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
)

func main() {
	var err error
	var c Config

	err = c.Parse("./config.yaml")
	if err != nil {
		klog.Error(err)
		os.Exit(1)
	}
	fmt.Println(c.Configmap)
	err = dev.DevInit(c.Configmap)
	if err != nil {
		klog.Error(err)
		os.Exit(1)
	}

	globals.MqttClient = mappercommon.MqttClient{IP: c.Mqtt.ServerAddress,
		User:   c.Mqtt.Username,
		Passwd: c.Mqtt.Password}
	klog.Info("Mqtt: ", c.Mqtt.ServerAddress, c.Mqtt.Username, c.Mqtt.Password)
	err = globals.MqttClient.Connect()
	if err != nil {
		klog.Error(err)
		os.Exit(1)
	}
	dev.DevStart()
}
