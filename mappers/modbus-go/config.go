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
	"errors"
	"io/ioutil"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
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
	PrivateKey    string `yaml:"privatekey,omitempty"`
}

// ErrConfigCert error of certification configuration.
var ErrConfigCert = errors.New("Both certification and private key must be provided")

var defaultConfigFile = "./config.yaml"

// Parse parse the configuration file. If failed, return error.
func (c *Config) Parse() error {
	var level klog.Level
	var loglevel string
	var configFile string

	pflag.StringVar(&loglevel, "v", "1", "log level")
	pflag.StringVar(&configFile, "config-file", defaultConfigFile, "Config file name")
	pflag.Parse()
	cf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err = yaml.Unmarshal(cf, c); err != nil {
		return err
	}
	if err = level.Set(loglevel); err != nil {
		return err
	}

	return c.parseFlags()
}

// parseFlags parse flags. Certification and Private key must be provided at the same time.
func (c *Config) parseFlags() error {
	pflag.StringVar(&c.Mqtt.ServerAddress, "mqtt-address", c.Mqtt.ServerAddress, "MQTT broker address")
	pflag.StringVar(&c.Mqtt.Username, "mqtt-username", c.Mqtt.Username, "username")
	pflag.StringVar(&c.Mqtt.Password, "mqtt-password", c.Mqtt.Password, "password")
	pflag.StringVar(&c.Mqtt.Cert, "mqtt-certification", c.Mqtt.Cert, "certification file path")
	pflag.StringVar(&c.Mqtt.PrivateKey, "mqtt-priviatekey", c.Mqtt.PrivateKey, "private key file path")
	pflag.Parse()

	if (c.Mqtt.Cert != "" && c.Mqtt.PrivateKey == "") ||
		(c.Mqtt.Cert == "" && c.Mqtt.PrivateKey != "") {
		return ErrConfigCert
	}
	return nil
}
