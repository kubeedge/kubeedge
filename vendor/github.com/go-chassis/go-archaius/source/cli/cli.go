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

//Package cli created on 2017/6/22.
package cli

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/go-chassis/go-archaius/source"
)

//const
const (
	Name                = "CommandlineSource"
	commandlinePriority = 2
)

//Source is source for all configuration
type Source struct {
	sync.RWMutex
	Configurations map[string]interface{}

	priority int
}

//NewCommandlineConfigSource defines a function used for creating configuration source
func NewCommandlineConfigSource() source.ConfigSource {
	cmdlineConfig := new(Source)
	cmdlineConfig.priority = commandlinePriority
	config := cmdlineConfig.pullCmdLineConfig()
	cmdlineConfig.Configurations = config

	return cmdlineConfig
}

func (*Source) pullCmdLineConfig() map[string]interface{} {
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

	return configMap
}

//GetConfigurations get configuration
func (cli *Source) GetConfigurations() (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	cli.Lock()
	defer cli.Unlock()
	for key, value := range cli.Configurations {
		configMap[key] = value
	}

	return configMap, nil
}

//GetConfigurationByKey gets required configuration for matching key
func (cli *Source) GetConfigurationByKey(key string) (interface{}, error) {
	cli.Lock()
	defer cli.Unlock()
	value, ok := cli.Configurations[key]
	if !ok {
		return nil, errors.New("key does not exist")
	}
	return value, nil
}

//GetPriority gets the priority of a configuration
func (cli *Source) GetPriority() int {
	return cli.priority
}

//SetPriority custom priority
func (cli *Source) SetPriority(priority int) {
	cli.priority = priority
}

//GetSourceName gets the source's name of a configuration
func (*Source) GetSourceName() string {
	return Name
}

//Cleanup cleans up a configuration
func (cli *Source) Cleanup() error {
	cli.Configurations = nil
	return nil
}

//Watch dynamically handles a configuration
func (*Source) Watch(callback source.EventHandler) error {
	return nil
}

//AddDimensionInfo  is none function
func (cli *Source) AddDimensionInfo(labels map[string]string) error {
	return nil
}

//Set no use
func (cli *Source) Set(key string, value interface{}) error {
	return nil
}

//Delete no use
func (cli *Source) Delete(key string) error {
	return nil
}
