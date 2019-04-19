package types

// DeviceProfile is structure to store in configMap
type DeviceProfile struct {
	DeviceInstances  []*DeviceInstance  `json:"deviceInstances,omitempty"`
	DeviceModels     []*DeviceModel     `json:"deviceModels,omitempty"`
	Protocols        []*Protocol        `json:"protocols,omitempty"`
	PropertyVisitors []*PropertyVisitor `json:"propertyVisitors,omitempty"`
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
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Properties  []*Property `json:"properties,omitempty"`
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
	ProtocolConfig interface{} `json:"protocol_config"`
}

// PropertyVisitor is structure to store propertyVisitor in deviceProfile.json in configmap
type PropertyVisitor struct {
	Name          string      `json:"name"`
	PropertyName  string      `json:"propertyName"`
	ModelName     string      `json:"modelName"`
	Protocol      string      `json:"protocol"`
	VisitorConfig interface{} `json:"visitorConfig"`
}
