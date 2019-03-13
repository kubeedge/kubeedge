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

package configuration

import (
	"io/ioutil"

	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/action_manager"
	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/data_converter"
	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/scheduler"
	"github.com/kubeedge/kubeedge/mappers/bluetooth_mapper/watcher"

	"gopkg.in/yaml.v2"
)

//CONFIGURE  contains the location of the configuration file
var CONFIGURE = "configuration/config.yaml"

//Config is the global configuration used by all the modules of the mapper
var Config *BLEConfig

//BLEConfig is the structure that stores the configuration information read from the config file
type BLEConfig struct {
	Mqtt          Mqtt                        `yaml:"mqtt"`
	Device        Device                      `yaml:"device"`
	Watcher       watcher.Watcher             `yaml:"watcher"`
	Scheduler     scheduler.Scheduler         `yaml:"scheduler"`
	ActionManager actionmanager.ActionManager `yaml:"action-manager"`
	Converter     dataconverter.Converter     `yaml:"data-converter"`
}

//Mqtt structure contains the MQTT specific configurations
type Mqtt struct {
	Mode           int    `yaml:"mode"`
	InternalServer string `yaml:"internal-server"`
	Server         string `yaml:"server"`
}

//Device structure contains the device specific configurations
type Device struct {
	Id   string `yaml:"id"`
	Name string `yaml:"name"`
}

//Load is used to load the information from the configuration file
func (b *BLEConfig) Load() error {
	yamlFile, err := ioutil.ReadFile(CONFIGURE)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, b)
	if err != nil {
		return err
	}
	Config = b
	return nil
}
