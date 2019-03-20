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

// Package core provides a list of interface for Dispatcher and ConfigMgr
package core

// Dispatcher is an interface for events Management
type Dispatcher interface {
	// Function to send Events to all listener
	DispatchEvent(event *Event) error
	//Function to Register all listener for different key changes
	RegisterListener(listenerObj EventListener, keys ...string) error
	// remove listener
	UnRegisterListener(listenerObj EventListener, keys ...string) error
}

// EventListener All EventListener should implement this Interface
type EventListener interface {
	Event(event *Event)
}

// ConfigMgr manager Source
type ConfigMgr interface {
	AddSource(source ConfigSource, priority int) error
	GetConfigurations() map[string]interface{}
	GetConfigurationsByDimensionInfo(dimensionInfo string) (map[string]interface{}, error)
	AddDimensionInfo(dimensionInfo string) (map[string]string, error)
	GetConfigurationsByKey(key string) interface{}
	GetConfigurationsByKeyAndDimensionInfo(dimensionInfo, key string) interface{}
	IsKeyExist(string) bool
	Unmarshal(interface{}) error
	Refresh(sourceName string) error
	Cleanup()
}

// ConfigSource should implement this interface
type ConfigSource interface {
	GetSourceName() string
	GetConfigurations() (map[string]interface{}, error)
	GetConfigurationsByDI(dimensionInfo string) (map[string]interface{}, error)
	GetConfigurationByKey(string) (interface{}, error)
	GetConfigurationByKeyAndDimensionInfo(key, dimensionInfo string) (interface{}, error)
	AddDimensionInfo(dimensionInfo string) (map[string]string, error)
	DynamicConfigHandler(DynamicConfigCallback) error
	GetPriority() int
	Cleanup() error
}

// Event generated when any config changes
type Event struct {
	EventSource string
	EventType   string
	Key         string
	Value       interface{}
}

// Event Constant
const (
	Update        = "UPDATE"
	Delete        = "DELETE"
	Create        = "CREATE"
	InvalidAction = "INVALID-ACTION"
)

// DynamicConfigCallback is an interface for creating event on object change
type DynamicConfigCallback interface {
	OnEvent(*Event)
}

// var SourcePool = make([]ConfigSource, 0)
