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
	"io/ioutil"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// Config is the modbus mapper configuration.
type Config struct {
	Mqtt      Mqtt   `yaml:"mqtt,omitempty"`
	Configmap string `yaml:"configmap"`
}

// Mqtt is the Mqtt configuration.
type Mqtt struct {
	ServerAddress string `yaml:"server,omitempty"`
	Username      string `yaml:"username,omitempty"`
	Password      string `yaml:"password,omitempty"`
	Cert          string `yaml:"certification,omitempty"`
}

// Parse parse the configuration file. If failed, return error.
func (c *Config) Parse(configFile string) error {
	cf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(cf, c)
	if err != nil {
		return err
	}
	c.parseFlags()
	return nil
}

func (c *Config) parseFlags() {
	pflag.StringVar(&c.Mqtt.ServerAddress, "mqtt-address", c.Mqtt.ServerAddress, "MQTT broker address")
	pflag.StringVar(&c.Mqtt.Username, "mqtt-username", c.Mqtt.Username, "username")
	pflag.StringVar(&c.Mqtt.Password, "mqtt-password", c.Mqtt.Password, "password")
	pflag.StringVar(&c.Mqtt.Cert, "mqtt-certification", c.Mqtt.Cert, "certification")
	pflag.Parse()
}
