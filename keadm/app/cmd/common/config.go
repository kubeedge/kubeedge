/*
Copyright 2019 The Kubeedge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

//Write2File writes data into a file in path
func Write2File(path string, data interface{}) error {
	d, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path, d, 0666); err != nil {
		return err
	}
	return nil
}

//WriteControllerYamlFile writes controller.yaml for cloud component
func WriteControllerYamlFile(path string, controllerData ControllerYaml) error {
	return Write2File(path, controllerData)
}

//WriteCloudModulesYamlFile writes modules.yaml for cloud component
func WriteCloudModulesYamlFile(path string, modulesData ModulesYaml) error {
	return Write2File(path, modulesData)
}

//WriteCloudLoggingYamlFile writes logging yaml for cloud component
func WriteCloudLoggingYamlFile(path string, loggingData LoggingYaml) error {
	return Write2File(path, loggingData)
}

//WriteEdgeLoggingYamlFile writes logging yaml for edge component
func WriteEdgeLoggingYamlFile(path string) error {
	loggingData := LoggingYaml{LoggerLevel: "DEBUG", EnableRsysLog: false, LogFormatText: true, Writers: []string{"stdout"}}
	if err := Write2File(path, loggingData); err != nil {
		return err
	}
	return nil
}

//WriteEdgeModulesYamlFile writes modules.yaml for edge component
func WriteEdgeModulesYamlFile(path string) error {
	modulesData := ModulesYaml{Modules: ModulesSt{Enabled: []string{"eventbus", "servicebus", "websocket", "metaManager", "edged", "twin", "edgemesh"}}}
	if err := Write2File(path, modulesData); err != nil {
		return err
	}
	return nil
}

//WriteEdgeYamlFile write conf/edge.yaml for edge component
func WriteEdgeYamlFile(path string, edgeData EdgeYamlSt) error {
	return Write2File(path, edgeData)
}
