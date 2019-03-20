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

//Package envconfigsource created on 2017/6/22.
package envconfigsource

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/go-chassis/go-archaius/core"
	"github.com/go-mesh/openlogging"
)

const (
	envSourceConst            = "EnvironmentSource"
	envVariableSourcePriority = 3
)

//EnvConfigurationSource is a struct
type EnvConfigurationSource struct {
	Configurations map[string]interface{}
	sync.RWMutex
}

var _ core.ConfigSource = &EnvConfigurationSource{}

var envConfigSource *EnvConfigurationSource

//NewEnvConfigurationSource configures a new environment configuration
func NewEnvConfigurationSource() core.ConfigSource {
	if envConfigSource == nil {
		envConfigSource = new(EnvConfigurationSource)
		config, err := envConfigSource.pullConfigurations()
		if err != nil {
			openlogging.GetLogger().Error("failed to initialize environment configurations: " + err.Error())
			return envConfigSource
		}
		envConfigSource.Configurations = config
	}

	return envConfigSource
}

func (*EnvConfigurationSource) pullConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	for _, value := range os.Environ() {
		rs := []rune(value)
		in := strings.Index(value, "=")
		configMap[string(rs[0:in])] = string(rs[in+1:])
	}

	return configMap, nil
}

//GetConfigurations gets all configuration
func (confSrc *EnvConfigurationSource) GetConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	confSrc.Lock()
	defer confSrc.Unlock()
	for key, value := range confSrc.Configurations {
		configMap[key] = value
	}

	return configMap, nil
}

//GetConfigurationByKey gets required configuration for a particular key
func (confSrc *EnvConfigurationSource) GetConfigurationByKey(key string) (interface{}, error) {
	confSrc.Lock()
	defer confSrc.Unlock()
	value, ok := confSrc.Configurations[key]
	if !ok {
		return nil, errors.New("key does not exist")
	}

	return value, nil
}

//GetPriority returns priority of environment configuration
func (*EnvConfigurationSource) GetPriority() int {
	return envVariableSourcePriority
}

//GetSourceName returns the name of environment source
func (*EnvConfigurationSource) GetSourceName() string {
	return envSourceConst
}

//DynamicConfigHandler dynamically handles a environment configuration
func (*EnvConfigurationSource) DynamicConfigHandler(callback core.DynamicConfigCallback) error {
	return nil
}

//GetConfigurationByKeyAndDimensionInfo gets a required environment configuration for particular key and dimension info pair
func (*EnvConfigurationSource) GetConfigurationByKeyAndDimensionInfo(key, di string) (interface{}, error) {
	return nil, nil
}

//Cleanup cleans a particular environment configuration up
func (confSrc *EnvConfigurationSource) Cleanup() error {
	confSrc.Configurations = nil
	return nil
}

//AddDimensionInfo adds dimension info for a environment configuration
func (*EnvConfigurationSource) AddDimensionInfo(dimensionInfo string) (map[string]string, error) {
	return nil, nil
}

//GetConfigurationsByDI gets required environment configuration for a particular dimension info
func (EnvConfigurationSource) GetConfigurationsByDI(dimensionInfo string) (map[string]interface{}, error) {
	return nil, nil
}
