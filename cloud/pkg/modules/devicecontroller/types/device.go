package types

// Device the struct of device
type Device struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	State       string              `json:"state,omitempty"`
	LastOnline  string              `json:"last_online,omitempty"`
	Attributes  map[string]*MsgAttr `json:"attributes,omitempty"`
	Twin        map[string]*MsgTwin `json:"twin,omitempty"`
}

// BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

// Parameter container para
type Parameter struct {
	EventID string
	Code    int
	Reason  string
}

// Result the struct of Result for sending
type Result struct {
	BaseMessage
	Code   int    `json:"code,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// MembershipDetail the struct of membership detail
type MembershipDetail struct {
	BaseMessage
	Devices []Device `json:"devices"`
}

// MembershipUpdate the struct of membership update
type MembershipUpdate struct {
	BaseMessage
	AddDevices    []Device `json:"added_devices"`
	RemoveDevices []Device `json:"removed_devices"`
}

// MsgAttr the struct of device attr
type MsgAttr struct {
	Value    string        `json:"value"`
	Optional *bool         `json:"optional,omitempty"`
	Metadata *TypeMetadata `json:"metadata,omitempty"`
}

// MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

// TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value,omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

// TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

// TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

// ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp,omitempty"`
}

// ConnectedInfo connected info
type ConnectedInfo struct {
	EventType string `json:"event_type"`
	TimeStamp int64  `json:"timestamp"`
}

// DeviceTwinDocument the struct of twin document
type DeviceTwinDocument struct {
	BaseMessage
	Twin map[string]*TwinDoc `json:"twin"`
}

// TwinDoc the struct of twin document
type TwinDoc struct {
	LastState    *MsgTwin `json:"last"`
	CurrentState *MsgTwin `json:"current"`
}

// DeviceTwinUpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}
