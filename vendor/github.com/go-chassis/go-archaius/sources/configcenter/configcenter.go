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

//Package configcenter created on 2017/6/22.
package configcenter

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-chassis-config"
	"github.com/go-mesh/openlogging"
)

const (
	//ConfigCenterSourceConst variable of type string
	ConfigCenterSourceConst    = "ConfigCenterSource"
	configCenterSourcePriority = 0
)

var (
	//ConfigPath is a variable of type string
	ConfigPath = ""
	//ConfigRefreshPath is a variable of type string
	ConfigRefreshPath = ""
)

//Handler handles configs from config center
type Handler struct {
	cc                           config.Client
	dynamicConfigHandler         *DynamicConfigHandler
	dimensionInfoMap             map[string]string
	Configurations               map[string]interface{}
	dimensionsInfoConfiguration  map[string]map[string]interface{}
	dimensionsInfoConfigurations []map[string]map[string]interface{}
	initSuccess                  bool
	connsLock                    sync.Mutex
	sync.RWMutex
	RefreshMode     int
	RefreshInterval time.Duration
	priority        int
}

//ConfigCenterConfig is pointer of config center source
var ConfigCenterConfig *Handler

//NewConfigCenterSource initializes all components of configuration center
func NewConfigCenterSource(cc config.Client,
	refreshMode, refreshInterval int) core.ConfigSource {
	if ConfigCenterConfig == nil {
		ConfigCenterConfig = new(Handler)
		ConfigCenterConfig.priority = configCenterSourcePriority
		ConfigCenterConfig.cc = cc
		ConfigCenterConfig.initSuccess = true
		ConfigCenterConfig.RefreshMode = refreshMode
		ConfigCenterConfig.RefreshInterval = time.Second * time.Duration(refreshInterval)

	}
	return ConfigCenterConfig
}

//GetConfigAPI is map
type GetConfigAPI map[string]map[string]interface{}

// ensure to implement config source
var _ core.ConfigSource = &Handler{}

//GetConfigurations gets a particular configuration
func (cfgSrcHandler *Handler) GetConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	err := cfgSrcHandler.refreshConfigurations("")
	if err != nil {
		return nil, err
	}
	if cfgSrcHandler.RefreshMode == 1 {
		go cfgSrcHandler.refreshConfigurationsPeriodically("")
	}

	cfgSrcHandler.Lock()
	for key, value := range cfgSrcHandler.Configurations {
		configMap[key] = value
	}
	cfgSrcHandler.Unlock()
	return configMap, nil
}

//GetConfigurationsByDI gets required configurations for particular dimension info
func (cfgSrcHandler *Handler) GetConfigurationsByDI(dimensionInfo string) (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	err := cfgSrcHandler.refreshConfigurations(dimensionInfo)
	if err != nil {
		return nil, err
	}

	if cfgSrcHandler.RefreshMode == 1 {
		go cfgSrcHandler.refreshConfigurationsPeriodically(dimensionInfo)
	}

	cfgSrcHandler.Lock()
	for key, value := range cfgSrcHandler.dimensionsInfoConfiguration {
		configMap[key] = value
	}
	cfgSrcHandler.Unlock()
	return configMap, nil
}

func (cfgSrcHandler *Handler) refreshConfigurationsPeriodically(dimensionInfo string) {
	ticker := time.Tick(cfgSrcHandler.RefreshInterval)
	isConnectionFailed := false
	for range ticker {
		err := cfgSrcHandler.refreshConfigurations(dimensionInfo)
		if err == nil {
			if isConnectionFailed {
				openlogging.GetLogger().Infof("Recover configurations from config center server")
			}
			isConnectionFailed = false
		} else {
			isConnectionFailed = true
		}
	}
}

func (cfgSrcHandler *Handler) refreshConfigurations(dimensionInfo string) error {
	var (
		config     map[string]interface{}
		configByDI map[string]map[string]interface{}
		err        error
		events     []*core.Event
	)

	if dimensionInfo == "" {
		config, err = cfgSrcHandler.cc.PullConfigs(cfgSrcHandler.cc.Options().ServiceName,
			cfgSrcHandler.cc.Options().Version, cfgSrcHandler.cc.Options().App, cfgSrcHandler.cc.Options().Env)
		if err != nil {
			openlogging.GetLogger().Warnf("Failed to pull configurations from config center server", err) //Warn
			return err
		}
		openlogging.Debug("pull configs", openlogging.WithTags(openlogging.Tags{
			"config": config,
		}))
		//Populate the events based on the changed value between current config and newly received Config
		events, err = cfgSrcHandler.populateEvents(config)
	} else {
		configByDI, err = cfgSrcHandler.cc.PullConfigsByDI(dimensionInfo)
		if err != nil {
			openlogging.GetLogger().Warnf("Failed to pull configurations from config center server", err) //Warn
			return err
		}
		//Populate the events based on the changed value between current config and newly received Config based dimension info
		events, err = cfgSrcHandler.setKeyValueByDI(configByDI, dimensionInfo)
	}

	if err != nil {
		openlogging.GetLogger().Warnf("error in generating event", err)
		return err
	}

	//Generate OnEvent Callback based on the events created
	if cfgSrcHandler.dynamicConfigHandler != nil {
		openlogging.GetLogger().Debugf("event On Receive %+v", events)
		for _, event := range events {
			cfgSrcHandler.dynamicConfigHandler.EventHandler.Callback.OnEvent(event)
		}
	}

	cfgSrcHandler.Lock()
	cfgSrcHandler.updateDimensionsInfoConfigurations(dimensionInfo, configByDI, config)
	cfgSrcHandler.Unlock()

	return nil
}

func (cfgSrcHandler *Handler) updateDimensionsInfoConfigurations(dimensionInfo string,
	configByDI map[string]map[string]interface{}, config map[string]interface{}) {

	if dimensionInfo == "" {
		cfgSrcHandler.Configurations = config

	} else {
		if len(cfgSrcHandler.dimensionsInfoConfigurations) != 0 {
			for _, j := range cfgSrcHandler.dimensionsInfoConfigurations {
				// This condition is used to add the information of dimension info if there are 2 dimension
				if len(j) == 0 {
					cfgSrcHandler.dimensionsInfoConfigurations = append(cfgSrcHandler.dimensionsInfoConfigurations, configByDI)
				}
				for p := range j {
					if (p != dimensionInfo && len(cfgSrcHandler.dimensionInfoMap) > len(cfgSrcHandler.dimensionsInfoConfigurations)) || (len(j) == 0) {
						cfgSrcHandler.dimensionsInfoConfigurations = append(cfgSrcHandler.dimensionsInfoConfigurations, configByDI)
					}
					_, ok := j[dimensionInfo]
					if ok {
						delete(j, dimensionInfo)
						cfgSrcHandler.dimensionsInfoConfigurations = append(cfgSrcHandler.dimensionsInfoConfigurations, configByDI)
					}
				}
			}
			// This for loop to remove the emty map "map[]" from cfgSrcHandler.dimensionsInfoConfigurations
			for i, v := range cfgSrcHandler.dimensionsInfoConfigurations {
				if len(v) == 0 && len(cfgSrcHandler.dimensionsInfoConfigurations) > 1 {
					cfgSrcHandler.dimensionsInfoConfigurations = append(cfgSrcHandler.dimensionsInfoConfigurations[:i], cfgSrcHandler.dimensionsInfoConfigurations[i+1:]...)
				}
			}
		} else {
			cfgSrcHandler.dimensionsInfoConfigurations = append(cfgSrcHandler.dimensionsInfoConfigurations, configByDI)
		}

	}
}

//GetConfigurationByKey gets required configuration for a particular key
func (cfgSrcHandler *Handler) GetConfigurationByKey(key string) (interface{}, error) {
	cfgSrcHandler.Lock()
	configSrcVal, ok := cfgSrcHandler.Configurations[key]
	cfgSrcHandler.Unlock()
	if ok {
		return configSrcVal, nil
	}

	return nil, errors.New("key not exist")
}

//GetConfigurationByKeyAndDimensionInfo gets required configuration for a particular key and dimension pair
func (cfgSrcHandler *Handler) GetConfigurationByKeyAndDimensionInfo(key, dimensionInfo string) (interface{}, error) {
	var (
		configSrcVal interface{}
		actualValue  interface{}
		exist        bool
	)

	cfgSrcHandler.Lock()
	for _, v := range cfgSrcHandler.dimensionsInfoConfigurations {
		value, ok := v[dimensionInfo]
		if ok {
			actualValue, exist = value[key]
		}
	}
	cfgSrcHandler.Unlock()

	if exist {
		configSrcVal = actualValue
		return configSrcVal, nil
	}

	return nil, errors.New("key not exist")
}

//AddDimensionInfo adds dimension info for a configuration
func (cfgSrcHandler *Handler) AddDimensionInfo(dimensionInfo string) (map[string]string, error) {
	if len(cfgSrcHandler.dimensionInfoMap) == 0 {
		cfgSrcHandler.dimensionInfoMap = make(map[string]string)
	}

	for i := range cfgSrcHandler.dimensionInfoMap {
		if i == dimensionInfo {
			openlogging.GetLogger().Errorf("dimension info already exist")
			return cfgSrcHandler.dimensionInfoMap, errors.New("dimension info allready exist")
		}
	}

	cfgSrcHandler.dimensionInfoMap[dimensionInfo] = dimensionInfo

	return cfgSrcHandler.dimensionInfoMap, nil
}

//GetSourceName returns name of the configuration
func (*Handler) GetSourceName() string {
	return ConfigCenterSourceConst
}

//GetPriority returns priority of a configuration
func (cfgSrcHandler *Handler) GetPriority() int {
	return cfgSrcHandler.priority
}

//SetPriority custom priority
func (cfgSrcHandler *Handler) SetPriority(priority int) {
	cfgSrcHandler.priority = priority
}

//DynamicConfigHandler dynamically handles a configuration
func (cfgSrcHandler *Handler) DynamicConfigHandler(callback core.DynamicConfigCallback) error {
	if cfgSrcHandler.initSuccess != true {
		return errors.New("config center source initialization failed")
	}

	dynCfgHandler, err := newDynConfigHandlerSource(cfgSrcHandler, callback)
	if err != nil {
		openlogging.GetLogger().Error("failed to initialize dynamic config center Handler:" + err.Error())
		return errors.New("failed to initialize dynamic config center Handler")
	}
	cfgSrcHandler.dynamicConfigHandler = dynCfgHandler

	if cfgSrcHandler.RefreshMode == 0 {
		// Pull All the configuration for the first time.
		cfgSrcHandler.refreshConfigurations("")
		//Start a web socket connection to receive change events.
		dynCfgHandler.startDynamicConfigHandler()
	}

	return nil
}

//Cleanup cleans the particular configuration up
func (cfgSrcHandler *Handler) Cleanup() error {
	cfgSrcHandler.connsLock.Lock()
	defer cfgSrcHandler.connsLock.Unlock()

	if cfgSrcHandler.dynamicConfigHandler != nil {
		cfgSrcHandler.dynamicConfigHandler.Cleanup()
	}

	cfgSrcHandler.dynamicConfigHandler = nil
	cfgSrcHandler.Configurations = nil

	return nil
}

func (cfgSrcHandler *Handler) populateEvents(updatedConfig map[string]interface{}) ([]*core.Event, error) {
	events := make([]*core.Event, 0)
	newConfig := make(map[string]interface{})
	cfgSrcHandler.Lock()
	defer cfgSrcHandler.Unlock()

	currentConfig := cfgSrcHandler.Configurations

	// generate create and update event
	for key, value := range updatedConfig {
		newConfig[key] = value
		currentValue, ok := currentConfig[key]
		if !ok { // if new configuration introduced
			events = append(events, cfgSrcHandler.constructEvent(core.Create, key, value))
		} else if !reflect.DeepEqual(currentValue, value) {
			events = append(events, cfgSrcHandler.constructEvent(core.Update, key, value))
		}
	}

	// generate delete event
	for key, value := range currentConfig {
		_, ok := newConfig[key]
		if !ok { // when old config not present in new config
			events = append(events, cfgSrcHandler.constructEvent(core.Delete, key, value))
		}
	}

	// update with latest config
	cfgSrcHandler.Configurations = newConfig

	return events, nil
}

func (cfgSrcHandler *Handler) setKeyValueByDI(updatedConfig map[string]map[string]interface{}, dimensionInfo string) ([]*core.Event, error) {
	events := make([]*core.Event, 0)
	newConfigForDI := make(map[string]map[string]interface{})
	cfgSrcHandler.Lock()
	defer cfgSrcHandler.Unlock()

	currentConfig := cfgSrcHandler.dimensionsInfoConfiguration

	// generate create and update event
	for key, value := range updatedConfig {
		if key == dimensionInfo {
			newConfigForDI[key] = value
			for k, v := range value {
				if len(currentConfig) == 0 {
					events = append(events, cfgSrcHandler.constructEvent(core.Create, k, v))
				}
				for diKey, val := range currentConfig {
					if diKey == dimensionInfo {
						currentValue, ok := val[k]
						if !ok { // if new configuration introduced
							events = append(events, cfgSrcHandler.constructEvent(core.Create, k, v))
						} else if currentValue != v {
							events = append(events, cfgSrcHandler.constructEvent(core.Update, k, v))
						}
					}
				}
			}
		}
	}

	// generate delete event
	for key, value := range currentConfig {
		if key == dimensionInfo {
			for k, v := range value {
				for _, val := range newConfigForDI {
					_, ok := val[k]
					if !ok {
						events = append(events, cfgSrcHandler.constructEvent(core.Delete, k, v))
					}
				}
			}
		}
	}

	// update with latest config
	cfgSrcHandler.dimensionsInfoConfiguration = newConfigForDI

	return events, nil
}

func (cfgSrcHandler *Handler) constructEvent(eventType string, key string, value interface{}) *core.Event {
	newEvent := new(core.Event)
	newEvent.EventSource = ConfigCenterSourceConst
	newEvent.EventType = eventType
	newEvent.Key = key
	newEvent.Value = value

	return newEvent
}
