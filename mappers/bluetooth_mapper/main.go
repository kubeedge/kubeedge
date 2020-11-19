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

package main

import (
	"flag"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/configuration"
	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/controller"
)

// main function
func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	BleConfig := configuration.BLEConfig{}
	// load config
	err := BleConfig.Load()
	if err != nil {
		klog.Errorf("Error in loading configuration: %s", err)
		os.Exit(1)
	}
	bleController := controller.Config{
		Watcher:       BleConfig.Watcher,
		ActionManager: BleConfig.ActionManager,
		Scheduler:     BleConfig.Scheduler,
		Converter:     BleConfig.Converter,
		Device:        BleConfig.Device,
		Mqtt:          BleConfig.Mqtt,
	}
	bleController.Start()
}
