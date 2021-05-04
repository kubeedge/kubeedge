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
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
)

//config.json decode struct
type Config struct {
	MqttEndpoint  string   `json:"mqttEndpoint"`
	TestManager   string   `json:"testManager"`
	EdgedEndpoint string   `json:"edgedEndpoint"`
	AppImageURL   []string `json:"image_url"`
	NodeID        string   `json:"nodeId"`
}

//config struct
var config *Config

//get config.json path
func LoadConfig() Config {
	if config == nil {
		config = loadConfigJSONFromPath()
	}
	return *config
}

//Load Config.json from the PWD, and decode the config.
func loadConfigJSONFromPath() *Config {
	path := getConfigPath()
	_, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		common.Infof("Failed to get Abs path: %v", err)
		panic(err)
	}
	config := &Config{}
	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}
	return config
}

//Get config path from Env or hard code the file path
func getConfigPath() string {
	path := os.Getenv("TESTCONFIG")
	if path == "" {
		path = "config.json"
	}
	return path
}

//function to Generate Random string
func GetRandomString(length int) string {
	str := "-0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}
