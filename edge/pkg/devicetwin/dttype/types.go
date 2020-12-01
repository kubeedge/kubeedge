package dttype

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
)

//Device the struct of device
type Device struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	State       string              `json:"state,omitempty"`
	LastOnline  string              `json:"last_online,omitempty"`
	Attributes  map[string]*MsgAttr `json:"attributes,omitempty"`
	Twin        map[string]*MsgTwin `json:"twin,omitempty"`
}

//BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

var ErrorUnmarshal = errors.New("Unmarshal update request body failed, please check the request")
var ErrorUpdate = errors.New("Update twin error, key:twin does not exist")
var ErrorKey = errors.New("The key of twin must only include upper or lowercase letters, number, english, and special letter - _ . , : / @ # and the length of key should be less than 128 bytes")
var ErrorValue = errors.New("The value of twin must only include upper or lowercase letters, number, english, and special letter - _ . , : / @ # and the length of value should be less than 512 bytes")

//SetEventID set event id
func (bs *BaseMessage) SetEventID(eventID string) {
	bs.EventID = eventID
}

//BuildBaseMessage build base msg
func BuildBaseMessage() BaseMessage {
	now := time.Now().UnixNano() / 1e6
	return BaseMessage{
		EventID:   uuid.New().String(),
		Timestamp: now}
}

//Parameter container para
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

//MembershipDetail the struct of membership detail
type MembershipDetail struct {
	BaseMessage
	Devices []Device `json:"devices"`
}

//MembershipUpdate the struct of membership update
type MembershipUpdate struct {
	BaseMessage
	AddDevices    []Device `json:"added_devices"`
	RemoveDevices []Device `json:"removed_devices"`
}

//MarshalMembershipUpdate marshal membership update
func MarshalMembershipUpdate(result MembershipUpdate) ([]byte, error) {
	for i := range result.AddDevices {
		if result.AddDevices[i].Twin != nil {
			for k, v := range result.AddDevices[i].Twin {
				if v.Metadata != nil && strings.Compare(v.Metadata.Type, "deleted") == 0 {
					result.AddDevices[i].Twin[k] = nil
				}
				v.ActualVersion = nil
				v.ExpectedVersion = nil
			}
		}
	}
	for i := range result.RemoveDevices {
		if result.RemoveDevices[i].Twin != nil {
			for k, v := range result.RemoveDevices[i].Twin {
				if v.Metadata != nil && strings.Compare(v.Metadata.Type, "deleted") == 0 {
					result.RemoveDevices[i].Twin[k] = nil
				}
				v.ActualVersion = nil
				v.ExpectedVersion = nil
			}
		}
	}
	resultJSON, err := json.Marshal(result)
	return resultJSON, err
}

//MsgAttr the struct of device attr
type MsgAttr struct {
	Value    string        `json:"value"`
	Optional *bool         `json:"optional,omitempty"`
	Metadata *TypeMetadata `json:"metadata,omitempty"`
}

//MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

//TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value,omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

//TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

//TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

//ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp,omitempty"`
}

//UpdateCloudVersion update cloud version
func (tv *TwinVersion) UpdateCloudVersion() {
	tv.CloudVersion = tv.CloudVersion + 1
}

//UpdateEdgeVersion update edge version while dealing edge update
func (tv *TwinVersion) UpdateEdgeVersion() {
	tv.EdgeVersion = tv.EdgeVersion + 1
}

//CompareWithCloud compare with cloud vershon while dealing cloud update req
func (tv TwinVersion) CompareWithCloud(tvCloud TwinVersion) bool {
	return tvCloud.EdgeVersion >= tv.EdgeVersion
}

//UpdateCloudVersion update cloud version
func UpdateCloudVersion(version string) (string, error) {
	var twinversion TwinVersion
	err := json.Unmarshal([]byte(version), &twinversion)
	if err != nil {
		return "", err
	}
	twinversion.UpdateCloudVersion()
	result, err := json.Marshal(twinversion)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

//UpdateEdgeVersion update Edge version
func UpdateEdgeVersion(version string) (string, error) {
	var twinversion TwinVersion
	err := json.Unmarshal([]byte(version), &twinversion)
	if err != nil {
		return "", err
	}
	twinversion.UpdateEdgeVersion()
	result, err := json.Marshal(twinversion)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

//CompareVersion compare cloud version
func CompareVersion(cloudversion string, edgeversion string) bool {
	var twincloudversion TwinVersion
	err := json.Unmarshal([]byte(cloudversion), &twincloudversion)
	if err != nil {
		return false
	}
	var twinedgeversion TwinVersion
	err1 := json.Unmarshal([]byte(edgeversion), &twinedgeversion)
	if err1 != nil {
		return false
	}
	return twinedgeversion.CompareWithCloud(twincloudversion)
}

// ConnectedInfo connected info
type ConnectedInfo struct {
	EventType string `json:"event_type"`
	TimeStamp int64  `json:"timestamp"`
}

// UnmarshalConnectedInfo unmarshal connected info
func UnmarshalConnectedInfo(payload []byte) (ConnectedInfo, error) {
	var connectedInfo ConnectedInfo
	err := json.Unmarshal(payload, &connectedInfo)
	if err != nil {
		return connectedInfo, err
	}
	return connectedInfo, nil
}

//DeviceTwinDocument the struct of twin document
type DeviceTwinDocument struct {
	BaseMessage
	Twin map[string]*TwinDoc `json:"twin"`
}

//TwinDoc the struct of twin document
type TwinDoc struct {
	LastState    *MsgTwin `json:"last"`
	CurrentState *MsgTwin `json:"current"`
}

//DeviceTwinUpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

// UnmarshalDeviceTwinDocument unmarshal device twin document
func UnmarshalDeviceTwinDocument(payload []byte) (*DeviceTwinDocument, error) {
	var deviceTwinUpdate DeviceTwinDocument
	err := json.Unmarshal(payload, &deviceTwinUpdate)
	if err != nil {
		return &deviceTwinUpdate, err
	}
	return &deviceTwinUpdate, nil
}

// UnmarshalDeviceTwinUpdate unmarshal device twin update
func UnmarshalDeviceTwinUpdate(payload []byte) (*DeviceTwinUpdate, error) {
	var deviceTwinUpdate DeviceTwinUpdate
	err := json.Unmarshal(payload, &deviceTwinUpdate)
	if err != nil {
		return &deviceTwinUpdate, ErrorUnmarshal
	}
	if deviceTwinUpdate.Twin == nil {
		return &deviceTwinUpdate, ErrorUpdate
	}
	for key, value := range deviceTwinUpdate.Twin {
		match := dtcommon.ValidateTwinKey(key)
		if !match {
			return &deviceTwinUpdate, ErrorKey
		}
		if value != nil {
			if value.Expected != nil {
				if value.Expected.Value != nil {
					if *value.Expected.Value != "" {
						match := dtcommon.ValidateTwinValue(*value.Expected.Value)
						if !match {
							return &deviceTwinUpdate, ErrorValue
						}
					}
				}
			}
			if value.Actual != nil {
				if value.Actual.Value != nil {
					if *value.Actual.Value != "" {
						match := dtcommon.ValidateTwinValue(*value.Actual.Value)
						if !match {
							return &deviceTwinUpdate, ErrorValue
						}
					}
				}
			}
		}
	}
	return &deviceTwinUpdate, nil
}

//DealTwinResult the result of dealing twin
type DealTwinResult struct {
	Add        []dtclient.DeviceTwin
	Delete     []dtclient.DeviceDelete
	Update     []dtclient.DeviceTwinUpdate
	Result     map[string]*MsgTwin
	SyncResult map[string]*MsgTwin
	Document   map[string]*TwinDoc
	Err        error
}

//DealAttrResult the result of dealing attr
type DealAttrResult struct {
	Add    []dtclient.DeviceAttr
	Delete []dtclient.DeviceDelete
	Update []dtclient.DeviceAttrUpdate
	Result map[string]*MsgAttr
	Err    error
}
