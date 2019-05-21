/*
Copyright 2019 The KubeEdge Authors.

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

package configuration

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/action_manager"
	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/data_converter"
	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/scheduler"
	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/watcher"

	"gopkg.in/yaml.v2"
)

//ConfigFilePath contains the location of the configuration file
var ConfigFilePath = "configuration/config.yaml"

//ConfigMapPath contains the location of the configuration file
var ConfigMapPath = "/opt/kubeedge/deviceProfile.json"

//Config is the global configuration used by all the modules of the mapper
var Config *BLEConfig

//Blutooth protocol name
const (
	ProtocolName string = "BLUETOOTH"
	READWRITE    string = "ReadWrite"
	READ         string = "ReadOnly"
)

//BLEConfig is the main structure that stores the configuration information read from both the config file as well as the config map
type BLEConfig struct {
	Mqtt          Mqtt                        `yaml:"mqtt"`
	Device        Device                      `yaml:"device"`
	Watcher       watcher.Watcher             `yaml:"watcher"`
	Scheduler     scheduler.Scheduler         `yaml:"scheduler"`
	ActionManager actionmanager.ActionManager `yaml:"action-manager"`
	Converter     dataconverter.Converter     `yaml:"data-converter"`
}

//ReadConfigFile is the structure that is used to read the config file to get configuration information from the user
type ReadConfigFile struct {
	Mqtt            Mqtt                `yaml:"mqtt"`
	DeviceModelName string              `yaml:"device-model-name"`
	ActionManager   ActionManagerConfig `yaml:"action-manager"`
	Watcher         watcher.Watcher     `yaml:"watcher"`
	Scheduler       scheduler.Scheduler `yaml:"scheduler"`
}

//ActionManagerConfig is a structure that contains a list of actions
type ActionManagerConfig struct {
	Actions []Action `yaml:"actions"`
}

//Action is structure to define a device action
type Action struct {
	//PerformImmediately signifies whether the action is to be performed by the action-manager immediately or not
	PerformImmediately bool `yaml:"perform-immediately" json:"perform-immediately"`
	//Name is the name of the Action
	Name string `yaml:"name" json:"name"`
	//PropertyName is the name of the property defined in the device CRD
	PropertyName string `yaml:"device-property-name" json:"device-property-name"`
}

//Mqtt structure contains the MQTT specific configurations
type Mqtt struct {
	Mode           int    `yaml:"mode"`
	InternalServer string `yaml:"internal-server"`
	Server         string `yaml:"server"`
}

//Device structure contains the device specific configurations
type Device struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

//ReadFromConfigFile is used to load the information from the configuration file
func (readConfigFile *ReadConfigFile) ReadFromConfigFile() error {
	yamlFile, err := ioutil.ReadFile(ConfigFilePath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, readConfigFile)
	if err != nil {
		return err
	}
	return nil
}

//Load is used to consolidate the information loaded from the configuration file and the configmaps
func (b *BLEConfig) Load() error {
	readConfigFile := ReadConfigFile{}
	readConfigMap := DeviceProfile{}
	err := readConfigFile.ReadFromConfigFile()
	if err != nil {
		return errors.New("Error while reading from configuration file " + err.Error())
	}
	err = readConfigMap.ReadFromConfigMap()
	if err != nil {
		return errors.New("Error while reading from config map " + err.Error())
	}
	b.Mqtt = readConfigFile.Mqtt
	b.Scheduler = readConfigFile.Scheduler
	b.Watcher = readConfigFile.Watcher
	// Assign device information obtained from config file
	for _, device := range readConfigMap.DeviceInstances {
		if strings.ToUpper(device.Model) == strings.ToUpper(readConfigFile.DeviceModelName) {
			b.Device.ID = device.ID
			b.Device.Name = device.Model
		}
	}
	// Assign information required by action manager
	for _, actionConfig := range readConfigFile.ActionManager.Actions {
		action := actionmanager.Action{}
		action.Name = actionConfig.Name
		action.PerformImmediately = actionConfig.PerformImmediately

		for _, propertyVisitor := range readConfigMap.PropertyVisitors {
			if strings.ToUpper(propertyVisitor.ModelName) == strings.ToUpper(b.Device.Name) && strings.ToUpper(propertyVisitor.PropertyName) == strings.ToUpper(actionConfig.PropertyName) && strings.ToUpper(propertyVisitor.Protocol) == ProtocolName {
				propertyVisitorBytes, err := json.Marshal(propertyVisitor.VisitorConfig)
				if err != nil {
					return errors.New("Error in marshalling data property visitor configuration: " + err.Error())
				}
				bluetoothPropertyVisitor := VisitorConfigBluetooth{}
				err = json.Unmarshal(propertyVisitorBytes, &bluetoothPropertyVisitor)
				if err != nil {
					return errors.New("Error in unmarshalling data property visitor configuration: " + err.Error())
				}
				action.Operation.CharacteristicUUID = bluetoothPropertyVisitor.CharacteristicUUID
				newBluetoothVisitorConfig := VisitorConfigBluetooth{}
				if !reflect.DeepEqual(bluetoothPropertyVisitor.BluetoothDataConverter, newBluetoothVisitorConfig.BluetoothDataConverter) {
					readAction := dataconverter.ReadAction{}
					readAction.ActionName = actionConfig.Name
					readAction.ConversionOperation.StartIndex = bluetoothPropertyVisitor.BluetoothDataConverter.StartIndex
					readAction.ConversionOperation.EndIndex = bluetoothPropertyVisitor.BluetoothDataConverter.EndIndex
					readAction.ConversionOperation.ShiftRight = bluetoothPropertyVisitor.BluetoothDataConverter.ShiftRight
					readAction.ConversionOperation.ShiftLeft = bluetoothPropertyVisitor.BluetoothDataConverter.ShiftLeft
					for _, readOperations := range bluetoothPropertyVisitor.BluetoothDataConverter.OrderOfOperations {
						readAction.ConversionOperation.OrderOfExecution = append(readAction.ConversionOperation.OrderOfExecution, readOperations.BluetoothOperationType)
						switch strings.ToUpper(readOperations.BluetoothOperationType) {
						case strings.ToUpper(BluetoothAdd):
							readAction.ConversionOperation.Add = readOperations.BluetoothOperationValue
						case strings.ToUpper(BluetoothSubtract):
							readAction.ConversionOperation.Subtract = readOperations.BluetoothOperationValue
						case strings.ToUpper(BluetoothMultiply):
							readAction.ConversionOperation.Multiply = readOperations.BluetoothOperationValue
						case strings.ToUpper(BluetoothDivide):
							readAction.ConversionOperation.Divide = readOperations.BluetoothOperationValue
						}
					}
					b.Converter.DataRead.Actions = append(b.Converter.DataRead.Actions, readAction)
				}
				if bluetoothPropertyVisitor.DataWriteToBluetooth != nil {
					writeAttribute := dataconverter.WriteAttribute{}
					writeAttribute.Operations = make(map[string]dataconverter.DataMap, 1)
					dataMap := dataconverter.DataMap{}
					dataMap.DataMapping = bluetoothPropertyVisitor.DataWriteToBluetooth
					writeAttribute.Operations[actionConfig.Name] = dataMap
					writeAttribute.Name = propertyVisitor.PropertyName
					b.Converter.DataWrite.Attributes = append(b.Converter.DataWrite.Attributes, writeAttribute)
				}
			}
		}
		for _, deviceModel := range readConfigMap.DeviceModels {
			if strings.ToUpper(deviceModel.Name) == strings.ToUpper(b.Device.Name) {
				for _, property := range deviceModel.Properties {
					if strings.ToUpper(property.Name) == strings.ToUpper(actionConfig.PropertyName) {
						if property.AccessMode == READWRITE {
							action.Operation.Action = "Write"
							if strings.ToUpper(property.DataType) == "INT" {
								value := string(int(property.DefaultValue.(float64)))
								action.Operation.Value = []byte(value)
							} else if strings.ToUpper(property.DataType) == "STRING" {
								for _, converterAttribute := range b.Converter.DataWrite.Attributes {
									if strings.ToUpper(converterAttribute.Name) == strings.ToUpper(actionConfig.PropertyName) {
										for operationName, dataMap := range converterAttribute.Operations {
											if action.Name == operationName {
												if _, ok := dataMap.DataMapping[property.DefaultValue.(string)]; ok {
													action.Operation.Value = dataMap.DataMapping[property.DefaultValue.(string)]
												}
											}
										}
									}
								}
							}
						} else if property.AccessMode == READ {
							action.Operation.Action = "Read"
						}
					}
				}
			}
		}
		b.ActionManager.Actions = append(b.ActionManager.Actions, action)
	}
	Config = b
	return nil
}
