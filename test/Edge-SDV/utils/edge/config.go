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

package edge

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kubeedge/kubeedge/test/Edge-SDV/utils/common"
)

type Config struct {
	MqttEndpoint string `json:"mqtt"`
	DeviceId     string `json:"deviceId"`
}

var loadedAddConfig *Config

func LoadConfig() Config {
	if loadedAddConfig == nil {
		loadedAddConfig = loadConfigJsonFromPath()
	}

	return *loadedAddConfig
}
func loadConfigJsonFromPath() *Config {
	path := configPath()
	_, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		common.InfoV6("Failed to Abs path :%v", err)
	}
	var config *Config = &Config{}
	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}
	return config
}

func configPath() string {
	path := os.Getenv("CONFIG")
	if path == "" {
		path = "config.json"
	}
	return path
}
