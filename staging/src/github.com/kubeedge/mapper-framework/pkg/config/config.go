/*
Copyright 2023 The KubeEdge Authors.

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

package config

import (
	"os"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

var defaultConfigFile = "./config.yaml"
var config *Config

// Config is the common mapper configuration.
type Config struct {
	GrpcServer GRPCServer `yaml:"grpc_server"`
	Common     Common     `yaml:"common"`
}

type GRPCServer struct {
	SocketPath string `yaml:"socket_path"`
}

type Common struct {
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	APIVersion   string `yaml:"api_version"`
	Protocol     string `yaml:"protocol"`
	Address      string `yaml:"address"`
	EdgeCoreSock string `yaml:"edgecore_sock"`
	HTTPPort     string `yaml:"http_port"`
}

// Parse the configuration file. If failed, return error.
func Parse() (c *Config, err error) {
	var level klog.Level
	var loglevel string
	var configFile string

	pflag.StringVar(&loglevel, "v", "1", "log level")
	pflag.StringVar(&configFile, "config-file", defaultConfigFile, "Config file name")
	pflag.Parse()

	if err = level.Set(loglevel); err != nil {
		return nil, err
	}

	c = &Config{}
	cf, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(cf, c); err != nil {
		return nil, err
	}

	config = c
	return c, nil
}

func Cfg() *Config {
	return config
}
