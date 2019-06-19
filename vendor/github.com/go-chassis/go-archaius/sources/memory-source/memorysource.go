package memoryconfigsource

import (
	"errors"
	"sync"

	"github.com/go-chassis/go-archaius/core"
)

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

//Package memoryconfigsource created on 2017/6/22.

const (
	memorySourceConst            = "MemorySource"
	memoryVariableSourcePriority = 1
)

//MemoryConfigurationSource is a struct
type MemoryConfigurationSource struct {
	Configurations map[string]interface{}
	callback       core.DynamicConfigCallback
	sync.RWMutex
	CallbackCheck chan bool
	ChanStatus    bool
}

//MemorySource is a interface
type MemorySource interface {
	core.ConfigSource
	AddKeyValue(string, interface{}) error
	DeleteKeyValue(string, interface{}) error
}

var _ core.ConfigSource = &MemoryConfigurationSource{}

var memoryConfigSource *MemoryConfigurationSource

//NewMemoryConfigurationSource initializes all necessary components for memory configuration
func NewMemoryConfigurationSource() MemorySource {
	if memoryConfigSource == nil {
		memoryConfigSource = new(MemoryConfigurationSource)
		memoryConfigSource.Configurations = make(map[string]interface{})
		memoryConfigSource.CallbackCheck = make(chan bool)
	}

	return memoryConfigSource
}

//GetConfigurations gets all memory configurations
func (confSrc *MemoryConfigurationSource) GetConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	confSrc.Lock()
	defer confSrc.Unlock()
	for key, value := range confSrc.Configurations {
		configMap[key] = value
	}

	return configMap, nil
}

//GetConfigurationByKey gets required memory configuration for a particular key
func (confSrc *MemoryConfigurationSource) GetConfigurationByKey(key string) (interface{}, error) {
	confSrc.Lock()
	defer confSrc.Unlock()
	value, ok := confSrc.Configurations[key]
	if !ok {
		return nil, errors.New("key does not exist")
	}

	return value, nil
}

//GetPriority returns priority of the memory configuration
func (*MemoryConfigurationSource) GetPriority() int {
	return memoryVariableSourcePriority
}

//GetSourceName returns name of memory configuration
func (*MemoryConfigurationSource) GetSourceName() string {
	return memorySourceConst
}

//DynamicConfigHandler dynamically handles a memory configuration
func (confSrc *MemoryConfigurationSource) DynamicConfigHandler(callback core.DynamicConfigCallback) error {
	confSrc.callback = callback
	confSrc.CallbackCheck <- true
	return nil
}

//AddKeyValue creates new configuration for corresponding key and value pair
func (confSrc *MemoryConfigurationSource) AddKeyValue(key string, value interface{}) error {
	if !confSrc.ChanStatus {
		<-confSrc.CallbackCheck
		confSrc.ChanStatus = true
	}

	event := new(core.Event)
	event.EventSource = confSrc.GetSourceName()
	event.Key = key
	event.Value = value

	confSrc.Lock()
	if _, ok := confSrc.Configurations[key]; !ok {
		event.EventType = core.Create
	} else {
		event.EventType = core.Update
	}

	confSrc.Configurations[key] = value
	confSrc.Unlock()

	if confSrc.callback != nil {
		confSrc.callback.OnEvent(event)
	}

	return nil
}

//DeleteKeyValue creates new configuration for corresponding key and value pair
func (confSrc *MemoryConfigurationSource) DeleteKeyValue(key string, value interface{}) error {
	if !confSrc.ChanStatus {
		<-confSrc.CallbackCheck
		confSrc.ChanStatus = true
	}

	event := new(core.Event)
	event.EventSource = confSrc.GetSourceName()
	event.Key = key
	event.Value = value

	confSrc.Lock()
	if _, ok := confSrc.Configurations[key]; ok {
		event.EventType = core.Delete
	} else {
		return nil
	}

	confSrc.Configurations[key] = value
	confSrc.Unlock()

	if confSrc.callback != nil {
		confSrc.callback.OnEvent(event)
	}

	return nil
}

//Cleanup cleans a particular memory configuration up
func (confSrc *MemoryConfigurationSource) Cleanup() error {
	confSrc.Configurations = nil

	return nil
}

//GetConfigurationByKeyAndDimensionInfo gets a required memory configuration for particular key and dimension info pair
func (*MemoryConfigurationSource) GetConfigurationByKeyAndDimensionInfo(key, di string) (interface{}, error) {
	return nil, nil
}

//AddDimensionInfo adds dimension info for a memory configuration
func (*MemoryConfigurationSource) AddDimensionInfo(dimensionInfo string) (map[string]string, error) {
	return nil, nil
}

//GetConfigurationsByDI gets required memory configuration for a particular dimension info
func (MemoryConfigurationSource) GetConfigurationsByDI(dimensionInfo string) (map[string]interface{}, error) {
	return nil, nil
}
