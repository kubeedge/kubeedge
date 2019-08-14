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
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/configuration"
	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/controller"
)

//usage is responsible for setting up the default settings of all defined command-line flags for log.
func usage() {
	flag.PrintDefaults()
	os.Exit(2)
}

//init for getting command line arguments for log
func init() {
	klog.InitFlags(nil)
	flag.Usage = usage
	// NOTE: This next line is key you have to call flag.Parse() for the command line
	// options or "flags" that are defined in the log module to be picked up.
	flag.Parse()
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}

// main function
func main() {
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
