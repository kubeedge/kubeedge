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

package v1beta1

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeviceSpec represents a single device instance.
type DeviceSpec struct {
	// Required: DeviceModelRef is reference to the device model used as a template
	// to create the device instance.
	DeviceModelRef *v1.LocalObjectReference `json:"deviceModelRef,omitempty"`
	// NodeName is a request to schedule this device onto a specific node. If it is non-empty,
	// the scheduler simply schedules this device onto that node, assuming that it fits
	// resource requirements.
	// +optional
	NodeName string `json:"nodeName,omitempty"`
	// List of properties which describe the device properties.
	// properties list item must be unique by properties.Name.
	// +optional
	Properties []DeviceProperty `json:"properties,omitempty"`
	// Required: The protocol configuration used to connect to the device.
	Protocol ProtocolConfig `json:"protocol,omitempty"`
	// List of methods of device.
	// methods list item must be unique by method.Name.
	// +optional
	Methods []DeviceMethod `json:"methods,omitempty"`
}

// DeviceStatus reports the device state and the desired/reported values of twin attributes.
type DeviceStatus struct {
	// A list of device twins containing desired/reported desired/reported values of twin properties.
	// Optional: A passive device won't have twin properties and this list could be empty.
	// +optional
	Twins []Twin `json:"twins,omitempty"`
	// Optional: The state of the device.
	// +optional
	State string `json:"state,omitempty"`
	// Optional: The last time the device was online.
	// +optional
	LastOnlineTime string `json:"lastOnlineTime,omitempty"`
	// Optional: whether be reported to the cloud
	// +optional
	ReportToCloud bool `json:"reportToCloud,omitempty"`
	// Optional: Define how frequent mapper will report the device status.
	// +optional
	ReportCycle int64 `json:"reportCycle,omitempty"`
}

// Twin provides a logical representation of control properties (writable properties in the
// device model). The properties can have a Desired state and a Reported state. The cloud configures
// the `Desired`state of a device property and this configuration update is pushed to the edge node.
// The mapper sends a command to the device to change this property value as per the desired state .
// It receives the `Reported` state of the property once the previous operation is complete and sends
// the reported state to the cloud. Offline device interaction in the edge is possible via twin
// properties for control/command operations.
type Twin struct {
	// Required: The property name for which the desired/reported values are specified.
	// This property should be present in the device model.
	PropertyName string `json:"propertyName,omitempty"`
	// Required: the reported property value.
	Reported TwinProperty `json:"reported,omitempty"`
	// The meaning of here is to indicate desired value of `deviceProperty.Desired`
	// that the mapper has received in current cycle.
	// Useful in cases that people want to check whether the mapper is working
	// appropriately and its internal status is up-to-date.
	// This value should be only updated by devicecontroller upstream.
	ObservedDesired TwinProperty `json:"observedDesired,omitempty"`
}

// TwinProperty represents the device property for which an Expected/Actual state can be defined.
type TwinProperty struct {
	// Required: The value for this property.
	Value string `json:"value,"`
	// Additional metadata like timestamp when the value was reported etc.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ProtocolConfig struct {
	// Unique protocol name
	// Required.
	ProtocolName string `json:"protocolName,omitempty"`
	// Any config data
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	ConfigData *CustomizedValue `json:"configData,omitempty"`
}

// DeviceMethod describes the specifics all the methods of the device.
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

// DeviceProperty describes the specifics all the properties of the device.
type DeviceProperty struct {
	// Required: The device property name to be accessed. It must be unique.
	// Note: If you need to use the built-in stream data processing function, you need to define Name as saveFrame or saveVideo
	Name string `json:"name,omitempty"`
	// The desired property value.
	Desired TwinProperty `json:"desired,omitempty"`
	// Visitors are intended to be consumed by device mappers which connect to devices
	// and collect data / perform actions on the device.
	// Required: Protocol relevant config details about the how to access the device property.
	Visitors VisitorConfig `json:"visitors,omitempty"`
	// Define how frequent mapper will report the value.
	// +optional
	ReportCycle int64 `json:"reportCycle,omitempty"`
	// Define how frequent mapper will collect from device.
	// +optional
	CollectCycle int64 `json:"collectCycle,omitempty"`
	// whether be reported to the cloud
	ReportToCloud bool `json:"reportToCloud,omitempty"`
	// PushMethod represents the protocol used to push data,
	// please ensure that the mapper can access the destination address.
	// +optional
	PushMethod *PushMethod `json:"pushMethod,omitempty"`
}

type PushMethod struct {
	// HTTP Push method configuration for http
	// +optional
	HTTP *PushMethodHTTP `json:"http,omitempty"`
	// MQTT Push method configuration for mqtt
	// +optional
	MQTT *PushMethodMQTT `json:"mqtt,omitempty"`
	// OTEL Push Method configuration for otel
	// +optional
	OTEL *PushMethodOTEL `json:"otel,omitempty"`
	// DBMethod represents the method used to push data to database,
	// please ensure that the mapper can access the destination address.
	// +optional
	DBMethod *DBMethodConfig `json:"dbMethod,omitempty"`
}

type PushMethodHTTP struct {
	// +optional
	HostName string `json:"hostName,omitempty"`
	// +optional
	Port int64 `json:"port,omitempty"`
	// +optional
	RequestPath string `json:"requestPath,omitempty"`
	// +optional
	Timeout int64 `json:"timeout,omitempty"`
}

type PushMethodMQTT struct {
	// broker address, like mqtt://127.0.0.1:1883
	// +optional
	Address string `json:"address,omitempty"`
	// publish topic for mqtt
	// +optional
	Topic string `json:"topic,omitempty"`
	// qos of mqtt publish param
	// +optional
	QoS int32 `json:"qos,omitempty"`
	// Is the message retained
	// +optional
	Retained bool `json:"retained,omitempty"`
}

type PushMethodOTEL struct {
	// the target endpoint URL the Exporter will connect to, like https://localhost:4318/v1/metrics
	EndpointURL string `protobuf:"bytes,1,opt,name=endpointURL,proto3" json:"endpointURL,omitempty"`
}

type DBMethodConfig struct {
	// method configuration for database
	// +optional
	Influxdb2 *DBMethodInfluxdb2 `json:"influxdb2,omitempty"`
	// +optional
	Redis *DBMethodRedis `json:"redis,omitempty"`
	// +optional
	TDEngine *DBMethodTDEngine `json:"TDEngine,omitempty"`
	// +optional
	Mysql *DBMethodMySQL `json:"mysql,omitempty"`
}

type DBMethodInfluxdb2 struct {
	// Config of influx database
	// +optional
	Influxdb2ClientConfig *Influxdb2ClientConfig `json:"influxdb2ClientConfig"`
	// config of device data when push to influx database
	// +optional
	Influxdb2DataConfig *Influxdb2DataConfig `json:"influxdb2DataConfig"`
}

type Influxdb2ClientConfig struct {
	// Url of influx database
	// +optional
	URL string `json:"url,omitempty"`
	// Org of the user in influx database
	// +optional
	Org string `json:"org,omitempty"`
	// Bucket of the user in influx database
	// +optional
	Bucket string `json:"bucket,omitempty"`
}

type Influxdb2DataConfig struct {
	// Measurement of the user data
	// +optional
	Measurement string `json:"measurement,omitempty"`
	// the tag of device data
	// +optional
	Tag map[string]string `json:"tag,omitempty"`
	// FieldKey of the user data
	// +optional
	FieldKey string `json:"fieldKey,omitempty"`
}

type DBMethodRedis struct {
	// RedisClientConfig of redis database
	// +optional
	RedisClientConfig *RedisClientConfig `json:"redisClientConfig,omitempty"`
}

type RedisClientConfig struct {
	// Addr of Redis database
	// +optional
	Addr string `json:"addr,omitempty"`
	// Db of Redis database
	// +optional
	DB int `json:"db,omitempty"`
	// Poolsize of Redis database
	// +optional
	Poolsize int `json:"poolsize,omitempty"`
	// MinIdleConns of Redis database
	// +optional
	MinIdleConns int `json:"minIdleConns,omitempty"`
}

type DBMethodTDEngine struct {
	// tdengineClientConfig of tdengine database
	// +optional
	TDEngineClientConfig *TDEngineClientConfig `json:"TDEngineClientConfig,omitempty"`
}
type TDEngineClientConfig struct {
	// addr of tdEngine database
	// +optional
	Addr string `json:"addr,omitempty"`
	// dbname of tdEngine database
	// +optional
	DBName string `json:"dbName,omitempty"`
}

type DBMethodMySQL struct {
	MySQLClientConfig *MySQLClientConfig `json:"mysqlClientConfig,omitempty"`
}

type MySQLClientConfig struct {
	// mysql address,like localhost:3306
	Addr string `protobuf:"bytes,1,opt,name=addr,proto3" json:"addr,omitempty"`
	// database name
	Database string `protobuf:"bytes,2,opt,name=database,proto3" json:"database,omitempty"`
	// user name
	UserName string `protobuf:"bytes,3,opt,name=userName,proto3" json:"userName,omitempty"`
}

type VisitorConfig struct {
	// Required: name of customized protocol
	ProtocolName string `json:"protocolName,omitempty"`
	// Required: The configData of customized protocol
	// +kubebuilder:validation:XPreserveUnknownFields
	ConfigData *CustomizedValue `json:"configData,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Device is the Schema for the devices API
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DeviceSpec   `json:"spec,omitempty"`
	Status            DeviceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeviceList contains a list of Device
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}

// CustomizedValue contains a map type data
// +kubebuilder:validation:Type=object
type CustomizedValue struct {
	Data map[string]interface{} `json:"data"`
}

// MarshalJSON implements the Marshaler interface.
func (in *CustomizedValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(in.Data)
}

// UnmarshalJSON implements the Unmarshaler interface.
func (in *CustomizedValue) UnmarshalJSON(data []byte) error {
	var out map[string]interface{}
	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}
	in.Data = out
	return nil
}

// DeepCopyInto implements the DeepCopyInto interface.
func (in *CustomizedValue) DeepCopyInto(out *CustomizedValue) {
	bytes, err := json.Marshal(in.Data)
	if err != nil {
		panic(err)
	}
	var clone map[string]interface{}
	err = json.Unmarshal(bytes, &clone)
	if err != nil {
		panic(err)
	}
	out.Data = clone
}

// DeepCopy implements the DeepCopy interface.
func (in *CustomizedValue) DeepCopy() *CustomizedValue {
	if in == nil {
		return nil
	}
	out := new(CustomizedValue)
	in.DeepCopyInto(out)
	return out
}
