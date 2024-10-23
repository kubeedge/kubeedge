/*
Copyright 2023 The KubeEdge Authors.

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

import "encoding/json"

// DeviceInstance is structure to store detailed information about the device in the mapper.
type DeviceInstance struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	ProtocolName string `json:"protocol,omitempty"`
	PProtocol    ProtocolConfig
	Model        string           `json:"model,omitempty"`
	Twins        []Twin           `json:"twins,omitempty"`
	Properties   []DeviceProperty `json:"properties,omitempty"`
	Methods      []DeviceMethod   `json:"methods,omitempty"`
	Status       DeviceStatus     `json:"status,omitempty"`
}

// DeviceModel is structure to store detailed information about the devicemodel in the mapper.
type DeviceModel struct {
	ID          string          `json:"id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Namespace   string          `json:"namespace,omitempty"`
	Description string          `json:"description,omitempty"`
	Properties  []ModelProperty `json:"properties,omitempty"`
}

// ModelProperty is structure to store deviceModel property.
type ModelProperty struct {
	Name        string `json:"name,omitempty"`
	DataType    string `json:"dataType,omitempty"`
	Description string `json:"description,omitempty"`
	AccessMode  string `json:"accessMode,omitempty"`
	Minimum     string `json:"minimum,omitempty"`
	Maximum     string `json:"maximum,omitempty"`
	Unit        string `json:"unit,omitempty"`
}

// ProtocolConfig is structure to store protocol information in device.
type ProtocolConfig struct {
	// Unique protocol name
	// Required.
	ProtocolName string `json:"protocolName,omitempty"`
	// Any config data
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	ConfigData json.RawMessage `json:"configData,omitempty"`
}

// DeviceMethod is structure to store method in device.
type DeviceMethod struct {
	// Required: The device method name to be accessed. It must be unique.
	Name string `json:"name,omitempty"`
	// Define the description of device method.
	// +optional
	Description string `json:"description,omitempty"`
	// PropertyNames are list of device properties that device methods can control.
	// Required: A device method can control multiple device properties.
	PropertyNames []string `json:"propertyNames,omitempty"`
}

// DeviceStatus is structure to store parameters for device status reporting.
type DeviceStatus struct {
	// whether be reported to the cloud
	ReportToCloud bool  `json:"reportToCloud,omitempty"`
	ReportCycle   int64 `json:"reportCycle,omitempty"`
}

// DeviceProperty is structure to store propertyVisitor in device.
type DeviceProperty struct {
	Name         string          `json:"name,omitempty"`
	PropertyName string          `json:"propertyName,omitempty"`
	ModelName    string          `json:"modelName,omitempty"`
	Protocol     string          `json:"protocol,omitempty"`
	Visitors     json.RawMessage `json:"visitorConfig"`
	// whether be reported to the cloud
	ReportToCloud bool             `json:"reportToCloud,omitempty"`
	CollectCycle  int64            `json:"collectCycle"`
	ReportCycle   int64            `json:"reportCycle,omitempty"`
	PushMethod    PushMethodConfig `json:"pushMethod,omitempty"`
	PProperty     ModelProperty
}

// PushMethodConfig is structure to store push config
type PushMethodConfig struct {
	MethodName   string          `json:"MethodName"`
	MethodConfig json.RawMessage `json:"MethodConfig"`
	DBMethod     DBMethodConfig  `json:"dbMethod,omitempty"`
}

// DBMethodConfig is structure to store database config
type DBMethodConfig struct {
	DBMethodName string   `json:"dbMethodName"`
	DBConfig     DBConfig `json:"dbConfig"`
}

type DBConfig struct {
	Influxdb2ClientConfig json.RawMessage `json:"influxdb2ClientConfig"`
	Influxdb2DataConfig   json.RawMessage `json:"influxdb2DataConfig"`
	RedisClientConfig     json.RawMessage `json:"redisClientConfig"`
	TDEngineClientConfig  json.RawMessage `json:"TDEngineClientConfig"`
	MySQLClientConfig     json.RawMessage `json:"mysqlClientConfig"`
}

// Metadata is the metadata for data.
type Metadata struct {
	Timestamp string `json:"timestamp,omitempty"`
	Type      string `json:"type,omitempty"`
}

// Twin is the set/get pair to one register.
type Twin struct {
	PropertyName    string `json:"propertyName,omitempty"`
	Property        *DeviceProperty
	ObservedDesired TwinProperty `json:"observedDesired,omitempty"`
	Reported        TwinProperty `json:"reported,omitempty"`
}

type TwinProperty struct {
	// Required: The value for this property.
	Value string `json:"value,"`
	// Additional metadata like timestamp when the value was reported etc.
	// +optional
	Metadata Metadata `json:"metadata,omitempty"`
}
