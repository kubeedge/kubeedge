/*
Copyright 2022 The KubeEdge Authors.

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
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-generator/pkg/common"
)

var defaultConfigFile = "./config.yaml"

// Config is the common mapper configuration.
type Config struct {
	GrpcServer GRPCServer `yaml:"grpc_server"`
	Common     Common     `yaml:"common"`
	DevInit    DevInit    `yaml:"dev_init"`
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
}

type DevInit struct {
	Mode      string `yaml:"mode"`
	Configmap string `yaml:"configmap"`
}

// Parse the configuration file. If failed, return error.
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

	switch c.DevInit.Mode {
	case common.DevInitModeConfigmap:
		if _, err := ioutil.ReadFile(c.DevInit.Configmap); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			c.DevInit.Configmap = strings.TrimSpace(os.Getenv("DEVICE_PROFILE"))
		}
		if strings.TrimSpace(c.DevInit.Configmap) == "" {
			return errors.New("can not parse configmap")
		}
	case common.DevInitModeRegister:
	case "": // if mode is nil, use meta server mode
		c.DevInit.Mode = common.DevInitModeRegister
		fallthrough
	default:
		return errors.New("unsupported dev init mode " + c.DevInit.Mode)
	}

	return nil
}
