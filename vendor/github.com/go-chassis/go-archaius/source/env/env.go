/*
 * Copyright 2017 Huawei Technologies Co., Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

//Package env created on 2017/6/22.
package env

import (
	"errors"
	"github.com/go-chassis/go-archaius/source"
	"os"
	"strings"
	"sync"

	"github.com/go-mesh/openlogging"
)

const (
	envSourceConst            = "EnvironmentSource"
	envVariableSourcePriority = 3
)

//Source is a struct
type Source struct {
	Configurations map[string]interface{}
	sync.RWMutex
	priority int
}

//NewEnvConfigurationSource configures a new environment configuration
func NewEnvConfigurationSource() source.ConfigSource {
	openlogging.Info("enable env source")
	envConfigSource := new(Source)
	envConfigSource.priority = envVariableSourcePriority
	config, err := envConfigSource.pullConfigurations()
	if err != nil {
		openlogging.GetLogger().Error("failed to initialize environment configurations: " + err.Error())
		return envConfigSource
	}
	envConfigSource.Configurations = config

	return envConfigSource
}

func (*Source) pullConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})
	for _, value := range os.Environ() {
		rs := []rune(value)
		in := strings.Index(value, "=")
		key := string(rs[0:in])
		value := string(rs[in+1:])
		envKey := strings.Replace(key, "_", ".", -1)
		configMap[key] = value
		configMap[envKey] = value

	}
	return configMap, nil
}

//GetConfigurations gets all configuration
func (es *Source) GetConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	es.Lock()
	defer es.Unlock()
	for key, value := range es.Configurations {
		configMap[key] = value
	}

	return configMap, nil
}

//GetConfigurationByKey gets required configuration for a particular key
func (es *Source) GetConfigurationByKey(key string) (interface{}, error) {
	es.Lock()
	defer es.Unlock()
	value, ok := es.Configurations[key]
	if !ok {
		return nil, errors.New("key does not exist")
	}

	return value, nil
}

//GetPriority returns priority of environment configuration
func (es *Source) GetPriority() int {
	return es.priority
}

//SetPriority custom priority
func (es *Source) SetPriority(priority int) {
	es.priority = priority
}

//GetSourceName returns the name of environment source
func (*Source) GetSourceName() string {
	return envSourceConst
}

//Watch dynamically handles a environment configuration
func (*Source) Watch(callback source.EventHandler) error {
	//TODO env change
	return nil
}

//Cleanup cleans a particular environment configuration up
func (es *Source) Cleanup() error {
	es.Configurations = nil
	return nil
}

//AddDimensionInfo no use
func (es *Source) AddDimensionInfo(labels map[string]string) error {
	return nil
}

//Set no use
func (es *Source) Set(key string, value interface{}) error {
	return nil
}

//Delete no use
func (es *Source) Delete(key string) error {
	return nil
}
