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
	"io/ioutil"
)

// Bluetooth Protocol Operation type
const (
	BluetoothAdd      string = "Add"
	BluetoothSubtract string = "Subtract"
	BluetoothMultiply string = "Multiply"
	BluetoothDivide   string = "Divide"
)

// DeviceProfile is structure to store in configMap
type DeviceProfile struct {
	DeviceInstances  []DeviceInstance  `json:"deviceInstances,omitempty"`
	DeviceModels     []DeviceModel     `json:"deviceModels,omitempty"`
	Protocols        []Protocol        `json:"protocols,omitempty"`
	PropertyVisitors []PropertyVisitor `json:"propertyVisitors,omitempty"`
}

// DeviceInstance is structure to store device in deviceProfile.json in configmap
type DeviceInstance struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Model    string `json:"model,omitempty"`
}

// DeviceModel is structure to store deviceModel in deviceProfile.json in configmap
type DeviceModel struct {
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Properties  []Property `json:"properties,omitempty"`
}

// Property is structure to store deviceModel property
type Property struct {
	Name         string      `json:"name,omitempty"`
	DataType     string      `json:"dataType,omitempty"`
	Description  string      `json:"description,omitempty"`
	AccessMode   string      `json:"accessMode,omitempty"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	Minimum      int64       `json:"minimum,omitempty"`
	Maximum      int64       `json:"maximum,omitempty"`
	Unit         string      `json:"unit,omitempty"`
}

// Protocol is structure to store protocol in deviceProfile.json in configmap
type Protocol struct {
	Name           string      `json:"name,omitempty"`
	Protocol       string      `json:"protocol,omitempty"`
	ProtocolConfig interface{} `json:"protocol_config,omitempty"`
}

// PropertyVisitor is structure to store propertyVisitor in deviceProfile.json in configmap
type PropertyVisitor struct {
	Name          string      `json:"name,omitempty"`
	PropertyName  string      `json:"propertyName,omitempty"`
	ModelName     string      `json:"modelName,omitempty"`
	Protocol      string      `json:"protocol,omitempty"`
	VisitorConfig interface{} `json:"visitorConfig,omitempty"`
}

// Common visitor configurations for bluetooth protocol
type VisitorConfigBluetooth struct {
	// Required: Unique ID of the corresponding operation
	CharacteristicUUID string `json:"characteristicUUID,omitempty"`
	// Responsible for converting the data coming from the platform into a form that is understood by the bluetooth device
	// For example: "ON":[1], "OFF":[0]
	//+optional
	DataWriteToBluetooth map[string][]byte `json:"dataWrite,omitempty"`
	// Responsible for converting the data being read from the bluetooth device into a form that is understandable by the platform
	//+optional
	BluetoothDataConverter BluetoothReadConverter `json:"dataConverter,omitempty"`
}

// Specifies the operations that may need to be performed to convert the data
type BluetoothReadConverter struct {
	// Required: Specifies the start index of the incoming byte stream to be considered to convert the data.
	// For example: start-index:2, end-index:3 concatenates the value present at second and third index of the incoming byte stream. If we want to reverse the order we can give it as start-index:3, end-index:2
	StartIndex int `json:"startIndex,omitempty"`
	// Required: Specifies the end index of incoming byte stream to be considered to convert the data
	// the value specified should be inclusive for example if 3 is specified it includes the third index
	EndIndex int `json:"endIndex,omitempty"`
	// Refers to the number of bits to shift left, if left-shift operation is necessary for conversion
	// +optional
	ShiftLeft uint `json:"shiftLeft,omitempty"`
	// Refers to the number of bits to shift right, if right-shift operation is necessary for conversion
	// +optional
	ShiftRight uint `json:"shiftRight,omitempty"`
	// Specifies in what order the operations(which are required to be performed to convert incoming data into understandable form) are performed
	//+optional
	OrderOfOperations []BluetoothOperations `json:"orderOfOperations,omitempty"`
}

// Specify the operation that should be performed to convert incoming data into understandable form
type BluetoothOperations struct {
	// Required: Specifies the operation to be performed to convert incoming data
	BluetoothOperationType string `json:"operationType,omitempty"`
	// Required: Specifies with what value the operation is to be performed
	BluetoothOperationValue float64 `json:"operationValue,omitempty"`
}

//ReadFromConfigMap is used to load the information from the configmaps that are provided from the cloud
func (deviceProfile *DeviceProfile) ReadFromConfigMap() error {
	jsonFile, err := ioutil.ReadFile(ConfigMapPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonFile, deviceProfile)
	if err != nil {
		return err
	}
	return nil
}
