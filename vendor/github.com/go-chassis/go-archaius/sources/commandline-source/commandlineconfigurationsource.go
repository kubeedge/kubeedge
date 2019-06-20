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

//Package commandlinesource created on 2017/6/22.
package commandlinesource

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/go-chassis/go-archaius/core"
	"github.com/go-mesh/openlogging"
)

const (
	commandLineConfigSourceConst = "CommandlineSource"
	commandlinePriority          = 2
)

var _ core.ConfigSource = &CommandLineConfigurationSource{}

//CommandLineConfigurationSource is source for all configuration
type CommandLineConfigurationSource struct {
	Configurations map[string]interface{}
	sync.RWMutex
}

var cmdlineConfig *CommandLineConfigurationSource

//NewCommandlineConfigSource defines a fucntion used for creating configuration source
func NewCommandlineConfigSource() core.ConfigSource {
	if cmdlineConfig == nil {
		cmdlineConfig = new(CommandLineConfigurationSource)
		config, err := cmdlineConfig.pullCmdLineConfig()
		if err != nil {
			openlogging.GetLogger().Error("failed to initialize commandline configurations:" + err.Error())
			return cmdlineConfig
		}
		cmdlineConfig.Configurations = config
	}

	return cmdlineConfig
}

func (*CommandLineConfigurationSource) pullCmdLineConfig() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	for i, value := range os.Args {
		if i == 0 {
			continue
		}
		in := strings.Index(value, "=")
		if (value[0] == '-') && (value[1] == ('-')) && (in >= 4) {
			rs := []rune(value)
			configMap[string(rs[2:in])] = string(rs[in+1:])
		} else if (value[0] == '-') && (in == 2) {
			rs := []rune(value)
			configMap[string(rs[1:in])] = string(rs[in+1:])
		}
	}

	return configMap, nil
}

//GetConfigurations gets particular configuration
func (confSrc *CommandLineConfigurationSource) GetConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	confSrc.Lock()
	defer confSrc.Unlock()
	for key, value := range confSrc.Configurations {
		configMap[key] = value
	}

	return configMap, nil
}

//GetConfigurationByKey gets required configuration for matching key
func (confSrc *CommandLineConfigurationSource) GetConfigurationByKey(key string) (interface{}, error) {
	confSrc.Lock()
	defer confSrc.Unlock()
	value, ok := confSrc.Configurations[key]
	if !ok {
		return nil, errors.New("key does not exist")
	}

	return value, nil
}

//GetPriority gets the priority of a configuration
func (*CommandLineConfigurationSource) GetPriority() int {
	return commandlinePriority
}

//GetSourceName gets the source's name of a configuration
func (*CommandLineConfigurationSource) GetSourceName() string {
	return commandLineConfigSourceConst
}

//DynamicConfigHandler dynamically handles a configuration
func (*CommandLineConfigurationSource) DynamicConfigHandler(callback core.DynamicConfigCallback) error {
	return nil
}

//GetConfigurationByKeyAndDimensionInfo gets a required configuration for particular key and dimension info
func (*CommandLineConfigurationSource) GetConfigurationByKeyAndDimensionInfo(key, di string) (interface{}, error) {
	return nil, nil
}

//Cleanup cleans up a configuration
func (confSrc *CommandLineConfigurationSource) Cleanup() error {
	confSrc.Configurations = nil
	return nil
}

//AddDimensionInfo adds dimension info to a configuration
func (*CommandLineConfigurationSource) AddDimensionInfo(dimensionInfo string) (map[string]string, error) {
	return nil, nil
}

//GetConfigurationsByDI gets reuqired configuration for a particular dimentsion info
func (CommandLineConfigurationSource) GetConfigurationsByDI(dimensionInfo string) (map[string]interface{}, error) {
	return nil, nil
}
