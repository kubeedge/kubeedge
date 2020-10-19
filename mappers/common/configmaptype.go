package mappercommon

import "encoding/json"

// DeviceProfile is structure to store in configMap
type DeviceProfile struct {
	DeviceInstances []DeviceInstance `json:"deviceInstances,omitempty"`
	DeviceModels    []DeviceModel    `json:"deviceModels,omitempty"`
	Protocols       []Protocol       `json:"protocols,omitempty"`
}

// DeviceInstance is structure to store device in deviceProfile.json in configmap
type DeviceInstance struct {
	ID               string `json:"id,omitempty"`
	Name             string `json:"name,omitempty"`
	ProtocolName     string `json:"protocol,omitempty"`
	PProtocol        Protocol
	Model            string            `json:"model,omitempty"`
	Twins            []Twin            `json:"twins,omitempty"`
	Datas            Data              `json:"data,omitempty"`
	PropertyVisitors []PropertyVisitor `json:"propertyVisitors,omitempty"`
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
	Name                 string          `json:"name,omitempty"`
	Protocol             string          `json:"protocol,omitempty"`
	ProtocolConfigs      ProtocolConfig  `json:"protocolConfig,omitempty"`
	ProtocolCommonConfig json.RawMessage `json:"protocolCommonConfig,omitempty"`
}

type ProtocolConfig struct {
	SlaveID int16 `json:"slaveID,omitempty"`
}

// PropertyVisitor is structure to store propertyVisitor in deviceProfile.json in configmap
type PropertyVisitor struct {
	Name          string `json:"name,omitempty"`
	PropertyName  string `json:"propertyName,omitempty"`
	ModelName     string `json:"modelName,omitempty"`
	CollectCycle  int64  `json:"collectCycle"`
	ReportCycle   int64  `json:"reportcycle,omitempty"`
	PProperty     Property
	Protocol      string          `json:"protocol,omitempty"`
	VisitorConfig json.RawMessage `json:"visitorConfig"`
}

type Data struct {
	Properties []DataProperty `json:"dataProperties,omitempty"`
	Topic      string         `json:"datatopic,omitempty"`
}

type DataProperty struct {
	Metadatas    Metadata `json:"metadata,omitempty"`
	PropertyName string   `json:"propertyName,omitempty"`
	PVisitor     *PropertyVisitor
}

type Metadata struct {
	Timestamp string `json:"timestamp,omitempty"`
	Type      string `json:"type,omitempty"`
}

type Twin struct {
	PropertyName string `json:"propertyName,omitempty"`
	PVisitor     *PropertyVisitor
	Desired      DesiredData  `json:"desired,omitempty"`
	Reported     ReportedData `json:"reported,omitempty"`
}

type DesiredData struct {
	Value     string   `json:"value,omitempty"`
	Metadatas Metadata `json:"metadata,omitempty"`
}

type ReportedData struct {
	Value     string   `json:"value,omitempty"`
	Metadatas Metadata `json:"metadata,omitempty"`
}
