package dttype

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
)

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

// Result the struct of Result for sending
type Result struct {
	BaseMessage
	Code   int    `json:"code,omitempty"`
	Reason string `json:"reason,omitempty"`
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

// UnmarshalDeviceTwinUpdate unmarshal device twin update解析结构体并且校验twin的值是否正确
func UnmarshalDeviceTwinUpdate(payload []byte) (*v1alpha2.Device, error) {
	device := &v1alpha2.Device{}
	err := json.Unmarshal(payload, device)
	if err != nil {
		return device, ErrorUnmarshal
	}
	if device.Status.Twins == nil {
		return device, ErrorUpdate
	}

	for _, value := range device.Status.Twins {
		match := dtcommon.ValidateTwinKey(value.PropertyName)
		if !match {
			return device, ErrorKey
		}

		if value.Desired.Value != "" {
			match := dtcommon.ValidateTwinValue(value.Desired.Value)
			if !match {
				return device, ErrorValue
			}
		}
		if value.Reported.Value != "" {
			match := dtcommon.ValidateTwinValue(value.Reported.Value)
			if !match {
				return device, ErrorValue
			}
		}
	}
	return device, nil
}

//DealTwinResult the result of dealing twin
type DealTwinResult struct {
	Add        []dtclient.DeviceTwin
	Delete     []dtclient.DeviceTwinPrimaryKey
	Update     []dtclient.DeviceTwinUpdate
	SyncResult v1alpha2.Device
	Err        error
}
