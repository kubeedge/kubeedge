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

// Package configmanager provides  functions to communicate to Config-Center
package configmanager

import (
	"errors"
	"reflect"
	"sync"

	"fmt"
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-archaius/sources/file-source"
	"github.com/go-mesh/openlogging"
)

const (
	//DefaultPriority gives the default priority
	DefaultPriority = -1
)

// ConfigurationManager is a struct to stores information about different config-center and their configuration
type ConfigurationManager struct {
	Sources          map[string]core.ConfigSource
	sourceMapMux     sync.RWMutex
	ConfigurationMap map[string]string
	configMapMux     sync.RWMutex
	dispatcher       core.Dispatcher
	//logger           *logger.ConfigClientLogger
}

var _ core.ConfigMgr = &ConfigurationManager{}

// NewConfigurationManager creates an object of ConfigurationManager
func NewConfigurationManager(dispatcher core.Dispatcher) core.ConfigMgr {
	configMgr := new(ConfigurationManager)
	configMgr.dispatcher = dispatcher
	configMgr.Sources = make(map[string]core.ConfigSource)
	configMgr.ConfigurationMap = make(map[string]string)
	//configMgr.logger = cLogger

	return configMgr
}

// Cleanup close and cleanup config manager channel
func (configMgr *ConfigurationManager) Cleanup() {
	// cleanup all dynamic handler
	configMgr.sourceMapMux.Lock()
	defer configMgr.sourceMapMux.Unlock()
	for _, source := range configMgr.Sources {
		if source.GetSourceName() == filesource.FileConfigSourceConst {
			source.Cleanup()
			delete(configMgr.Sources, source.GetSourceName())
		}
	}
}

// Unmarshal deserailize config into object
func (configMgr *ConfigurationManager) Unmarshal(obj interface{}) error {
	rv := reflect.ValueOf(obj)
	// only pointers are accepted
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		err := errors.New("invalid object supplied")
		openlogging.GetLogger().Error("invalid object supplied: " + err.Error())
		return err
	}

	return configMgr.unmarshal(rv, doNotConsiderTag)
}

// AddSource adds a source to configurationManager
func (configMgr *ConfigurationManager) AddSource(source core.ConfigSource, priority int) error {
	if source == nil || source.GetSourceName() == "" {
		err := errors.New("nil or invalid source supplied")
		openlogging.GetLogger().Error("nil or invalid source supplied: " + err.Error())
		return err
	}

	configMgr.sourceMapMux.Lock()
	sourceName := source.GetSourceName()

	_, ok := configMgr.Sources[sourceName]
	if ok {
		err := errors.New("duplicate source supplied")
		openlogging.GetLogger().Error("duplicate source supplied: " + err.Error())
		configMgr.sourceMapMux.Unlock()
		return err
	}

	configMgr.Sources[sourceName] = source
	configMgr.sourceMapMux.Unlock()

	err := configMgr.pullSourceConfigs(sourceName)
	if err != nil {
		err = fmt.Errorf("fail to load configuration of %s source: %s", sourceName, err)
		openlogging.Error(err.Error())
		return err
	}
	openlogging.Info("invoke dynamic handler:" + source.GetSourceName())
	go source.DynamicConfigHandler(configMgr)

	return nil
}

func (configMgr *ConfigurationManager) pullSourceConfigs(source string) error {
	configMgr.sourceMapMux.Lock()
	configSource, ok := configMgr.Sources[source]
	configMgr.sourceMapMux.Unlock()
	if !ok {
		err := errors.New("invalid source or source not added")
		openlogging.GetLogger().Error("invalid source or source not added: " + err.Error())
		return err
	}

	config, err := configSource.GetConfigurations()
	if config == nil || len(config) == 0 {
		if err != nil {
			openlogging.GetLogger().Error("Get configuration by items failed: " + err.Error())
			return err
		}

		openlogging.GetLogger().Warnf("empty configurtion from %s", source)
		return nil
	}

	configMgr.updateConfigurationMap(configSource, config)

	return nil
}

func (configMgr *ConfigurationManager) pullSourceConfigsByDI(source, di string) error {
	configMgr.sourceMapMux.Lock()
	configSource, ok := configMgr.Sources[source]
	configMgr.sourceMapMux.Unlock()
	if !ok {
		err := errors.New("invalid source or source not addeded")
		openlogging.GetLogger().Error("invalid source or source not addeded: " + err.Error())
		return err
	}

	config, err := configSource.GetConfigurationsByDI(di)
	if config == nil || len(config) == 0 {
		if err != nil {
			openlogging.GetLogger().Error("Get configuration by items failed: " + err.Error())
			return err
		}

		openlogging.GetLogger().Warnf("empty configuration from %s", source)
		return nil
	}

	configMgr.updateConfigurationMapByDI(configSource, config)

	return nil
}

// GetConfigurations returns all the configurationkeys
func (configMgr *ConfigurationManager) GetConfigurations() map[string]interface{} {
	config := make(map[string]interface{}, 0)

	configMgr.configMapMux.Lock()
	defer configMgr.configMapMux.Unlock()

	for key, sourceName := range configMgr.ConfigurationMap {
		sValue := configMgr.configValueBySource(key, sourceName)
		if sValue == nil {
			continue
		}
		config[key] = sValue
	}

	return config
}

// GetConfigurationsByDimensionInfo returns list of all the configuration for a particular dimensionInfo
func (configMgr *ConfigurationManager) GetConfigurationsByDimensionInfo(dimensionInfo string) (map[string]interface{}, error) {
	config := make(map[string]interface{}, 0)

	configMgr.configMapMux.Lock()
	defer configMgr.configMapMux.Unlock()

	for key, sourceName := range configMgr.ConfigurationMap {
		sValue := configMgr.configValueBySourceAndDimensionInfo(key, sourceName, dimensionInfo)
		if sValue == nil {
			continue
		}
		config[key] = sValue
	}

	return config, nil
}

// AddDimensionInfo adds the dimensionInfo to the list of which configurations needs to be pulled
func (configMgr *ConfigurationManager) AddDimensionInfo(dimensionInfo string) (map[string]string, error) {
	config := make(map[string]string, 0)

	config, er := configMgr.addDimensionInfo(dimensionInfo)
	if er != nil {
		openlogging.GetLogger().Errorf("failed to do add dimension info %s", er)
		return config, er
	}

	err := configMgr.pullSourceConfigsByDI("ConfigCenterSource", dimensionInfo)
	if err != nil {
		openlogging.GetLogger().Errorf("fail to load configuration of ConfigCenterSource source%s", err)
		return nil, err
	}

	return config, nil
}

// Refresh refreshes the full configurations of all the dimnesionInfos
func (configMgr *ConfigurationManager) Refresh(sourceName string) error {
	err := configMgr.pullSourceConfigs(sourceName)
	if err != nil {
		openlogging.GetLogger().Errorf("fail to load configuration of %s source: %s", sourceName, err)
		errorMsg := "fail to load configuration of" + sourceName + " source"
		return errors.New(errorMsg)
	}
	return nil
}

func (configMgr *ConfigurationManager) configValueBySource(configKey, sourceName string) interface{} {
	configMgr.sourceMapMux.Lock()
	source, ok := configMgr.Sources[sourceName]
	configMgr.sourceMapMux.Unlock()
	if !ok {
		return nil
	}

	configValue, err := source.GetConfigurationByKey(configKey)
	if err != nil {
		// may be before getting config, Event has deleted it so get next priority config value
		nbSource := configMgr.findNextBestSource(configKey, sourceName)
		if nbSource != nil {
			configValue, _ := nbSource.GetConfigurationByKey(configKey)
			return configValue
		}
		return nil
	}

	return configValue
}

func (configMgr *ConfigurationManager) configValueBySourceAndDimensionInfo(configKey, sourceName, dimensionInfo string) interface{} {
	configMgr.sourceMapMux.Lock()
	source, ok := configMgr.Sources[sourceName]
	configMgr.sourceMapMux.Unlock()
	if !ok {
		return nil
	}

	configValue, err := source.GetConfigurationByKeyAndDimensionInfo(configKey, dimensionInfo)
	if err != nil {
		// getting config by dimension info is only for config-center so no need to get the next best source.
		return nil
	}

	return configValue
}

func (configMgr *ConfigurationManager) addDimensionInfo(dimensionInfo string) (map[string]string, error) {

	configMgr.sourceMapMux.Lock()
	source, ok := configMgr.Sources["ConfigCenterSource"]
	configMgr.sourceMapMux.Unlock()
	if !ok {
		openlogging.GetLogger().Errorf("source doesnot exist")
		return nil, errors.New("source doesnot exist")
	}

	config, err := source.AddDimensionInfo(dimensionInfo)
	return config, err
}

// IsKeyExist check if key exsist in cache
func (configMgr *ConfigurationManager) IsKeyExist(key string) bool {
	configMgr.configMapMux.Lock()
	defer configMgr.configMapMux.Unlock()

	if _, ok := configMgr.ConfigurationMap[key]; ok {
		return true
	}

	return false
}

// GetConfigurationsByKey returns the value for a particluar key from cache
func (configMgr *ConfigurationManager) GetConfigurationsByKey(key string) interface{} {
	configMgr.configMapMux.Lock()
	sourceName, ok := configMgr.ConfigurationMap[key]
	configMgr.configMapMux.Unlock()
	if !ok {
		return nil
	}

	return configMgr.configValueBySource(key, sourceName)
}

// GetConfigurationsByKeyAndDimensionInfo returns the key value for a particular dimensionInfo
func (configMgr *ConfigurationManager) GetConfigurationsByKeyAndDimensionInfo(dimensionInfo, key string) interface{} {
	configMgr.configMapMux.Lock()
	sourceName, ok := configMgr.ConfigurationMap[key]

	configMgr.configMapMux.Unlock()
	if !ok {
		return nil
	}

	return configMgr.configValueBySourceAndDimensionInfo(key, sourceName, dimensionInfo)
}

func (configMgr *ConfigurationManager) updateConfigurationMap(source core.ConfigSource, configs map[string]interface{}) error {
	configMgr.configMapMux.Lock()
	defer configMgr.configMapMux.Unlock()
	for key := range configs {
		sourceName, ok := configMgr.ConfigurationMap[key]
		if !ok { // if key do not exist then add source
			configMgr.ConfigurationMap[key] = source.GetSourceName()
			continue
		}

		configMgr.sourceMapMux.Lock()
		currentSource, ok := configMgr.Sources[sourceName]
		configMgr.sourceMapMux.Unlock()
		if !ok {
			configMgr.ConfigurationMap[key] = source.GetSourceName()
			continue
		}

		currentSrcPriority := currentSource.GetPriority()
		if currentSrcPriority > source.GetPriority() { // lesser value has high priority
			configMgr.ConfigurationMap[key] = source.GetSourceName()
		}
	}

	return nil
}

func (configMgr *ConfigurationManager) updateConfigurationMapByDI(source core.ConfigSource, configs map[string]interface{}) error {
	configMgr.configMapMux.Lock()
	defer configMgr.configMapMux.Unlock()
	for key := range configs {
		sourceName, ok := configMgr.ConfigurationMap[key]
		if !ok { // if key do not exist then add source
			configMgr.ConfigurationMap[key] = source.GetSourceName()
			continue
		}

		configMgr.sourceMapMux.Lock()
		currentSource, ok := configMgr.Sources[sourceName]
		configMgr.sourceMapMux.Unlock()
		if !ok {
			configMgr.ConfigurationMap[key] = source.GetSourceName()
			continue
		}

		currentSrcPriority := currentSource.GetPriority()
		if currentSrcPriority > source.GetPriority() { // lesser value has high priority
			configMgr.ConfigurationMap[key] = source.GetSourceName()
		}
	}

	return nil
}

func (configMgr *ConfigurationManager) updateEvent(event *core.Event) error {
	// refresh all configuration one by one
	if event == nil || event.EventSource == "" || event.Key == "" {
		return errors.New("nil or invalid event supplied")
	}

	openlogging.GetLogger().Debugf("EventReceived %s", event)
	//log.Println("EventReceived", event)
	switch event.EventType {
	case core.Create, core.Update:
		configMgr.configMapMux.Lock()
		sourceName, ok := configMgr.ConfigurationMap[event.Key]
		if !ok {
			configMgr.ConfigurationMap[event.Key] = event.EventSource
			event.EventType = core.Create
		} else if sourceName == event.EventSource {
			event.EventType = core.Update
		} else if sourceName != event.EventSource {
			prioritySrc := configMgr.getHighPrioritySource(sourceName, event.EventSource)
			if prioritySrc != nil && prioritySrc.GetSourceName() == sourceName {
				// if event generated from less priority source then ignore
				configMgr.configMapMux.Unlock()
				return nil
			}
			configMgr.ConfigurationMap[event.Key] = event.EventSource
			event.EventType = core.Update
		}
		configMgr.configMapMux.Unlock()

	case core.Delete:
		configMgr.configMapMux.Lock()
		sourceName, ok := configMgr.ConfigurationMap[event.Key]
		if !ok || sourceName != event.EventSource {
			// if delete event generated from source not maintained ignore it
			configMgr.configMapMux.Unlock()
			return nil
		} else if sourceName == event.EventSource {
			// find less priority source or delete key
			source := configMgr.findNextBestSource(event.Key, sourceName)
			if source == nil {
				delete(configMgr.ConfigurationMap, event.Key)
			} else {
				configMgr.ConfigurationMap[event.Key] = source.GetSourceName()
			}
		}
		configMgr.configMapMux.Unlock()
	}

	configMgr.dispatcher.DispatchEvent(event)

	return nil
}

// OnEvent Triggers actions when an event is generated
func (configMgr *ConfigurationManager) OnEvent(event *core.Event) {
	err := configMgr.updateEvent(event)
	if err != nil {
		openlogging.GetLogger().Error("failed in updating event with error: " + err.Error())
	}
}

func (configMgr *ConfigurationManager) findNextBestSource(key string, sourceName string) core.ConfigSource {
	var rSource core.ConfigSource
	configMgr.sourceMapMux.Lock()
	for _, source := range configMgr.Sources {
		if source.GetSourceName() == sourceName {
			continue
		}
		value, err := source.GetConfigurationByKey(key)
		if err != nil || value == nil {
			continue
		}
		if rSource == nil {
			rSource = source
			continue
		}
		if source.GetPriority() < rSource.GetPriority() { // less value has high priority
			rSource = source
		}
	}
	configMgr.sourceMapMux.Unlock()

	return rSource
}

func (configMgr *ConfigurationManager) getHighPrioritySource(srcNameA, srcNameB string) core.ConfigSource {
	configMgr.sourceMapMux.Lock()
	sourceA, okA := configMgr.Sources[srcNameA]
	sourceB, okB := configMgr.Sources[srcNameB]
	configMgr.sourceMapMux.Unlock()

	if !okA && !okB {
		return nil
	} else if !okA {
		return sourceB
	} else if !okB {
		return sourceA
	}

	if sourceA.GetPriority() < sourceB.GetPriority() { //less value has high priority
		return sourceA
	}

	return sourceB
}
