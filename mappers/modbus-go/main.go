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
	"os"

	"k8s.io/klog/v2"

	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus-go/device"
	"github.com/kubeedge/kubeedge/mappers/modbus-go/globals"
)

func main() {
	var err error
	var config Config

	klog.InitFlags(nil)
	defer klog.Flush()

	if err = config.Parse(); err != nil {
		klog.Fatal(err)
		os.Exit(1)
	}
	klog.V(4).Info(config.Configmap)

	globals.MqttClient = mappercommon.MqttClient{IP: config.Mqtt.ServerAddress,
		User:       config.Mqtt.Username,
		Passwd:     config.Mqtt.Password,
		Cert:       config.Mqtt.Cert,
		PrivateKey: config.Mqtt.PrivateKey}
	if err = globals.MqttClient.Connect(); err != nil {
		klog.Fatal(err)
		os.Exit(1)
	}

	if err = device.DevInit(config.Configmap); err != nil {
		klog.Fatal(err)
		os.Exit(1)
	}
	device.DevStart()
}
