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
package utils

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type vmSpec struct {
	IP       string `json:"ip"`
	Username string `json:"username"`
	Passwd   string `json:"password"`
}

//config.json decode struct
type Config struct {
	AppImageURL                    []string          `json:"image_url"`
	K8SMasterForKubeEdge           string            `json:"k8smasterforkubeedge"`
	Nodes                          map[string]vmSpec `json:"k8snodes"`
	NumOfNodes                     int               `json:"node_num"`
	ImageRepo                      string            `json:"imagerepo"`
	K8SMasterForProvisionEdgeNodes string            `json:"k8smasterforprovisionedgenodes"`
	CloudImageURL                  string            `json:"cloudimageurl"`
	EdgeImageURL                   string            `json:"edgeimageurl"`
	Namespace                      string            `json:"namespace"`
	ControllerStubPort             int               `json:"controllerstubport"`
	Protocol                       string            `json:"protocol"`
	DockerHubUserName              string            `json:"dockerhubusername"`
	DockerHubPassword              string            `json:"dockerhubpassword"`
	MqttEndpoint                   string            `json:"mqttendpoint"`
	KubeConfigPath                 string            `json:"kubeconfigpath"`
	Token                          string            `json:"token"`
}

//config struct
var config *Config

//get config.json path
func LoadConfig() Config {
	if config == nil {
		config = loadConfigJSOMFromPath()
	}
	return *config
}

//loadConfigJSOMFromPath reads the test configuration and builds a Config object.
func loadConfigJSOMFromPath() *Config {
	path := getConfigPath()
	_, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		Infof("Failed to get Abs path: %v", err)
		panic(err)
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

//getConfigPath returns the configuration path provided in the env var name. In case the env var is not
//set, the default configuration path is returned
func getConfigPath() string {
	path := os.Getenv("TESTCONFIG")
	if path == "" {
		path = "config.json"
	}
	return path
}

func RandomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

//function to Generate Random string
func GetRandomString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}
