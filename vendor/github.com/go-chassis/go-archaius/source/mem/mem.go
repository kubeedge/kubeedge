package mem

import (
	"errors"
	"github.com/go-chassis/go-archaius/event"
	"sync"

	"github.com/go-chassis/go-archaius/source"
	"github.com/go-mesh/openlogging"
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

// const
const (
	Name                         = "MemorySource"
	memoryVariableSourcePriority = 1
)

//Source is a struct
type Source struct {
	sync.RWMutex
	Configurations map[string]interface{}

	callback source.EventHandler

	CallbackCheck chan bool
	ChanStatus    bool
	priority      int
}

//NewMemoryConfigurationSource initializes all necessary components for memory configuration
func NewMemoryConfigurationSource() source.ConfigSource {
	memoryConfigSource := new(Source)
	memoryConfigSource.priority = memoryVariableSourcePriority
	memoryConfigSource.Configurations = make(map[string]interface{})
	memoryConfigSource.CallbackCheck = make(chan bool)
	return memoryConfigSource
}

//GetConfigurations gets all memory configurations
func (ms *Source) GetConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	ms.Lock()
	defer ms.Unlock()
	for key, value := range ms.Configurations {
		configMap[key] = value
	}

	return configMap, nil
}

//GetConfigurationByKey gets required memory configuration for a particular key
func (ms *Source) GetConfigurationByKey(key string) (interface{}, error) {
	ms.Lock()
	defer ms.Unlock()
	value, ok := ms.Configurations[key]
	if !ok {
		return nil, errors.New("key does not exist")
	}

	return value, nil
}

//GetPriority returns priority of the memory configuration
func (ms *Source) GetPriority() int {
	return ms.priority
}

//SetPriority custom priority
func (ms *Source) SetPriority(priority int) {
	ms.priority = priority
}

//GetSourceName returns name of memory configuration
func (*Source) GetSourceName() string {
	return Name
}

//Watch dynamically handles a memory configuration
func (ms *Source) Watch(callback source.EventHandler) error {
	ms.callback = callback
	openlogging.Info("mem source callback prepared")
	ms.CallbackCheck <- true
	return nil
}

//Cleanup cleans a particular memory configuration up
func (ms *Source) Cleanup() error {
	ms.Configurations = nil

	return nil
}

//AddDimensionInfo  is none function
func (ms *Source) AddDimensionInfo(labels map[string]string) error {
	return nil
}

//Set set mem config
func (ms *Source) Set(key string, value interface{}) error {
	if !ms.ChanStatus {
		<-ms.CallbackCheck
		ms.ChanStatus = true
	}

	e := new(event.Event)
	e.EventSource = ms.GetSourceName()
	e.Key = key
	e.Value = value

	ms.Lock()
	defer ms.Unlock()
	if _, ok := ms.Configurations[key]; !ok {
		e.EventType = event.Create
	} else {
		e.EventType = event.Update
	}

	ms.Configurations[key] = value

	if ms.callback != nil {
		ms.callback.OnEvent(e)
	}

	return nil
}

//Delete remvove mem config
func (ms *Source) Delete(key string) error {
	if !ms.ChanStatus {
		<-ms.CallbackCheck
		ms.ChanStatus = true
	}

	e := new(event.Event)
	e.EventSource = ms.GetSourceName()
	e.Key = key

	ms.Lock()
	if v, ok := ms.Configurations[key]; ok {
		e.EventType = event.Delete
		e.Value = v
	} else {
		return nil
	}

	ms.Unlock()

	if ms.callback != nil {
		ms.callback.OnEvent(e)
	}

	return nil
}
