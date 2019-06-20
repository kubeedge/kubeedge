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

/*
* Created by on 2017/6/22.
 */

// Package goarchaius provides you a list of interface which helps in communciation with config-center
package goarchaius

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-archaius/core/cast"
	"github.com/go-chassis/go-archaius/core/config-manager"
	"github.com/go-chassis/go-archaius/core/event-system"
	"github.com/go-chassis/go-archaius/sources/commandline-source"
	"github.com/go-chassis/go-archaius/sources/enviromentvariable-source"
	"github.com/go-chassis/go-archaius/sources/memory-source"
	"github.com/go-mesh/openlogging"
)

const (
	//UnsuccessfulArchaiusInit is of type string
	UnsuccessfulArchaiusInit = "issue with go-archaius initialization"
)

// ConfigurationFactory is a list of Interface for Config Center
type ConfigurationFactory interface {
	// Init ConfigurationFactory
	Init() error
	// dump complete configuration managed by config-client based on priority
	// (1. Config Center 2. Commandline Argument 3.Environment Variable  4.ConfigFile , 1 with highest priority
	GetConfigurations() map[string]interface{}
	// dump complete configuration managed by config-client for Config Center based on dimension info.
	GetConfigurationsByDimensionInfo(dimensionInfo string) map[string]interface{}
	// add the dimension info for other services
	AddByDimensionInfo(dimensionInfo string) (map[string]string, error)
	// return all values of different sources
	GetConfigurationByKey(key string) interface{}
	// check for existence of key
	IsKeyExist(string) bool
	// unmarshal data on user define structure
	Unmarshal(structure interface{}) error
	// Add custom sources
	AddSource(core.ConfigSource) error
	//Function to Register all listener for different key changes, each key could be a regular expression
	RegisterListener(listenerObj core.EventListener, key ...string) error
	// remove listener
	UnRegisterListener(listenerObj core.EventListener, key ...string) error
	// DeInit
	DeInit() error
	// an abstraction to return key's value in respective type
	GetValue(key string) cast.Value
	// return values of config-center source based on key and dimension info
	GetConfigurationByKeyAndDimensionInfo(dimensionInfo, key string) interface{}
	// an abstraction to return key's value in respective type based on dimension info which is provided by user
	GetValueByDI(dimensionInfo, key string) cast.Value
}

// ConfigFactory is a struct which stores configuration information
type ConfigFactory struct {
	dispatcher  core.Dispatcher
	configMgr   core.ConfigMgr
	initSuccess bool
	//logger      *ccLogger.ConfigClientLogger
}

var arc *ConfigFactory

// NewConfigFactory creates a new configuration object for Config center
func NewConfigFactory(log openlogging.Logger) (ConfigurationFactory, error) {

	if arc == nil {

		arc = new(ConfigFactory)
		//// Source init should be before config manager init
		//sources.NewSourceInit()
		arc.dispatcher = eventsystem.NewDispatcher()
		arc.configMgr = configmanager.NewConfigurationManager(arc.dispatcher)

		// Default config source init
		// 1. Command line source
		cmdSource := commandlinesource.NewCommandlineConfigSource()
		arc.configMgr.AddSource(cmdSource, cmdSource.GetPriority())

		// Environment variable source
		envSource := envconfigsource.NewEnvConfigurationSource()
		arc.configMgr.AddSource(envSource, envSource.GetPriority())
		// External variable source
		memorySource := memoryconfigsource.NewMemoryConfigurationSource()
		arc.configMgr.AddSource(memorySource, memorySource.GetPriority())
	}

	return arc, nil
}

// Init intiates the Configurationfatory
func (arc *ConfigFactory) Init() error {
	arc.initSuccess = true
	return nil
}

// GetConfigurations dump complete configuration managed by config-client
//   Only return highest priority key value:-
//   1. ConfigFile 		2. Environment Variable
//   3. Commandline Argument	4. Config Center configuration
//   config-center value being the highest priority
func (arc *ConfigFactory) GetConfigurations() map[string]interface{} {
	if arc.initSuccess == false {
		return nil
	}

	return arc.configMgr.GetConfigurations()
}

//GetConfigurationsByDimensionInfo dump complete configuration managed by config-client Only return Config Center configurations.
func (arc *ConfigFactory) GetConfigurationsByDimensionInfo(dimensionInfo string) map[string]interface{} {
	if arc.initSuccess == false {
		return nil
	}

	config, err := arc.configMgr.GetConfigurationsByDimensionInfo(dimensionInfo)
	if err != nil {
		openlogging.GetLogger().Errorf("Failed to get the configuration by dimension info: %s", err)
	}

	return config
}

// AddByDimensionInfo adds a NewDimensionInfo of which configurations needs to be taken
func (arc *ConfigFactory) AddByDimensionInfo(dimensionInfo string) (map[string]string, error) {
	if arc.initSuccess == false {
		return nil, nil
	}

	config, err := arc.configMgr.AddDimensionInfo(dimensionInfo)
	return config, err
}

// GetConfigurationByKey return all values of different sources
func (arc *ConfigFactory) GetConfigurationByKey(key string) interface{} {
	if arc.initSuccess == false {
		return nil
	}

	return arc.configMgr.GetConfigurationsByKey(key)
}

// GetConfigurationByKeyAndDimensionInfo get the value for a key in a particular dimensionInfo
func (arc *ConfigFactory) GetConfigurationByKeyAndDimensionInfo(dimensionInfo, key string) interface{} {
	if arc.initSuccess == false {
		return nil
	}

	return arc.configMgr.GetConfigurationsByKeyAndDimensionInfo(dimensionInfo, key)
}

// AddSource return all values of different sources
func (arc *ConfigFactory) AddSource(source core.ConfigSource) error {
	if arc.initSuccess == false {
		return nil
	}

	return arc.configMgr.AddSource(source, source.GetPriority())
}

// IsKeyExist check existence of key
func (arc *ConfigFactory) IsKeyExist(key string) bool {
	if arc.initSuccess == false {
		return false
	}

	return arc.configMgr.IsKeyExist(key)
}

// RegisterListener Function to Register all listener for different key changes
func (arc *ConfigFactory) RegisterListener(listenerObj core.EventListener, keys ...string) error {
	for _, key := range keys {
		_, err := regexp.Compile(key)
		if err != nil {
			openlogging.GetLogger().Error(fmt.Sprintf("invalid key format for %s key. key registration ignored: %s", key, err))
			return fmt.Errorf("invalid key format for %s key", key)
		}
	}

	return arc.dispatcher.RegisterListener(listenerObj, keys...)
}

// UnRegisterListener remove listener
func (arc *ConfigFactory) UnRegisterListener(listenerObj core.EventListener, keys ...string) error {
	for _, key := range keys {
		_, err := regexp.Compile(key)
		if err != nil {
			openlogging.GetLogger().Error(fmt.Sprintf("invalid key format for %s key. key registration ignored: %s", key, err))
			return fmt.Errorf("invalid key format for %s key", key)
		}
	}

	return arc.dispatcher.UnRegisterListener(listenerObj, keys...)
}

// Unmarshal function is used in the case when user want his yaml file to be unmarshalled to structure pointer
// Unmarshal function accepts a pointer and in called function anyone can able to get the data in passed object
// Unmarshal only accepts a pointer values
// Unmarshal returns error if obj values are 0. nil and value type.
// Procedure:
//      1. Unmarshal first checks the passed object type using reflection.
//      2. Based on type Unmarshal function will check and set the values
//      ex: If type is basic types like int, string, float then it will assigb directly values,
//          If type is map, ptr and struct then it will again send for unmarshal untill it find the basic type and set the values
func (arc *ConfigFactory) Unmarshal(obj interface{}) error {
	if arc.initSuccess == false {
		return nil
	}

	return arc.configMgr.Unmarshal(obj)
}

// DeInit return all values of different sources
func (arc *ConfigFactory) DeInit() error {
	if arc.initSuccess == false {
		return nil
	}

	arc.configMgr.Cleanup()
	return nil
}

// GetValue an abstraction to return key's value in respective type
func (arc *ConfigFactory) GetValue(key string) cast.Value {
	if arc.initSuccess == false {
		return nil
	}

	var confValue cast.Value
	val := arc.GetConfigurationByKey(key)
	if val == nil {
		confValue = cast.NewValue(nil, errors.New("key does not exist"))
	} else {
		confValue = cast.NewValue(val, nil)
	}

	return confValue
}

// GetValueByDI an abstraction to return key's value in respective type based on dimension info which is provided by user
func (arc *ConfigFactory) GetValueByDI(dimensionInfo, key string) cast.Value {
	if arc.initSuccess == false {
		return nil
	}

	var confValue cast.Value
	val := arc.GetConfigurationByKeyAndDimensionInfo(dimensionInfo, key)
	if val == nil {
		confValue = cast.NewValue(nil, errors.New("key does not exist"))
	} else {
		confValue = cast.NewValue(val, nil)
	}

	return confValue
}
