package dttype

import (
	"encoding/json"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

// UnmarshalMembershipDetail Unmarshal membershipdetail
func UnmarshalMembershipDetail(payload []byte) (*MembershipDetail, error) {
	var membershipDetail MembershipDetail
	err := json.Unmarshal(payload, &membershipDetail)
	if err != nil {
		return nil, err
	}
	return &membershipDetail, nil
}

// UnmarshalMembershipUpdate Unmarshal membershipupdate
func UnmarshalMembershipUpdate(payload []byte) (*MembershipUpdate, error) {
	var membershipUpdate MembershipUpdate
	err := json.Unmarshal(payload, &membershipUpdate)
	if err != nil {
		return nil, err
	}
	return &membershipUpdate, nil
}

// UnmarshalBaseMessage Unmarshal get
func UnmarshalBaseMessage(payload []byte) (*BaseMessage, error) {
	var get BaseMessage
	err := json.Unmarshal(payload, &get)
	if err != nil {
		return nil, err
	}
	return &get, nil
}

// DeviceAttrToMsgAttr  deviceattr to msgattr
func DeviceAttrToMsgAttr(deviceAttrs []models.DeviceAttr) map[string]*MsgAttr {
	msgAttrs := make(map[string]*MsgAttr, len(deviceAttrs))
	for _, attr := range deviceAttrs {
		optional := attr.Optional
		msgAttrs[attr.Name] = &MsgAttr{
			Value:    attr.Value,
			Optional: &optional,
			Metadata: &TypeMetadata{Type: attr.AttrType}}
	}
	return msgAttrs
}

// DeviceTwinToMsgTwin  devicetwin contains meta and version to msgtwin,
func DeviceTwinToMsgTwin(deviceTwins []models.DeviceTwin) map[string]*MsgTwin {
	msgTwins := make(map[string]*MsgTwin, len(deviceTwins))
	for _, twin := range deviceTwins {
		var expectedMeta ValueMetadata
		var actualMeta ValueMetadata
		var expectedVersion TwinVersion
		var actualVersion TwinVersion

		optional := twin.Optional
		expected := twin.Expected
		actual := twin.Actual

		msgTwin := &MsgTwin{
			Optional: &optional,
			Metadata: &TypeMetadata{Type: twin.AttrType}}
		if expected != "" {
			expectedValue := &TwinValue{Value: &expected}
			if twin.ExpectedMeta != "" {
				if err := json.Unmarshal([]byte(twin.ExpectedMeta), &expectedMeta); err != nil {
					// TODO: handle error
					klog.Error(err)
				}
				expectedValue.Metadata = &expectedMeta
			}
			msgTwin.Expected = expectedValue
		}
		if actual != "" {
			actualValue := &TwinValue{Value: &actual}
			if twin.ActualMeta != "" {
				if err := json.Unmarshal([]byte(twin.ActualMeta), &actualMeta); err != nil {
					// TODO: handle error
					klog.Error(err)
				}
				actualValue.Metadata = &actualMeta
			}
			msgTwin.Actual = actualValue
		}

		if twin.ExpectedVersion != "" {
			if err := json.Unmarshal([]byte(twin.ExpectedVersion), &expectedVersion); err != nil {
				// TODO: handle error
				klog.Error(err)
			}
			msgTwin.ExpectedVersion = &expectedVersion
		}
		if twin.ActualVersion != "" {
			if err := json.Unmarshal([]byte(twin.ActualVersion), &actualVersion); err != nil {
				// TODO: handle error
				klog.Error(err)
			}
			msgTwin.ActualVersion = &actualVersion
		}
		msgTwins[twin.Name] = msgTwin
	}
	return msgTwins
}

// MsgAttrToDeviceAttr msgattr to deviceattr
func MsgAttrToDeviceAttr(name string, msgAttr *MsgAttr) models.DeviceAttr {
	attrType := "string"
	if msgAttr.Metadata != nil {
		attrType = msgAttr.Metadata.Type
	}
	optional := true
	if msgAttr.Optional != nil {
		optional = *msgAttr.Optional
	}
	return models.DeviceAttr{
		Name:     name,
		AttrType: attrType,
		Optional: optional}
}

// CopyMsgTwin copy msg twin
func CopyMsgTwin(msgTwin *MsgTwin, noVersion bool) MsgTwin {
	var result MsgTwin
	payload, _ := json.Marshal(msgTwin)
	if err := json.Unmarshal(payload, &result); err != nil {
		// TODO: handle error
		klog.Error(err)
	}
	if noVersion {
		result.ActualVersion = nil
		result.ExpectedVersion = nil
	}
	return result
}

// CopyMsgAttr copy msg attr
func CopyMsgAttr(msgAttr *MsgAttr) MsgAttr {
	var result MsgAttr
	payload, _ := json.Marshal(msgAttr)
	if err := json.Unmarshal(payload, &result); err != nil {
		// TODO: handle error
		klog.Error(err)
	}
	return result
}

// MsgTwinToDeviceTwin msgtwin convert to devicetwin
func MsgTwinToDeviceTwin(name string, msgTwin *MsgTwin) models.DeviceTwin {
	optional := true
	if msgTwin.Optional != nil {
		optional = *msgTwin.Optional
	}
	attrType := "string"
	if msgTwin.Metadata != nil {
		attrType = msgTwin.Metadata.Type
	}
	return models.DeviceTwin{
		Name:     name,
		AttrType: attrType,
		Optional: optional}
}

// DeviceMsg the struct of device state msg
type DeviceMsg struct {
	BaseMessage
	DeviceCloudMsg DeviceCloudMsg `json:"device"`
}

// BuildDeviceCloudMsgState build the msg
func BuildDeviceCloudMsgState(baseMessage BaseMessage, device Device) ([]byte, error) {
	result := DeviceMsg{
		BaseMessage: baseMessage,
		DeviceCloudMsg: DeviceCloudMsg{
			Name:           device.Name,
			State:          device.State,
			LastOnlineTime: device.LastOnline}}
	payload, err := json.Marshal(result)
	if err != nil {
		return []byte(""), err
	}
	return payload, nil
}

// DeviceAttrUpdate the struct of device attr update msg
type DeviceAttrUpdate struct {
	BaseMessage
	Attributes map[string]*MsgAttr `json:"attributes"`
}

// BuildDeviceAttrUpdate build the DeviceAttrUpdate
func BuildDeviceAttrUpdate(baseMessage BaseMessage, attrs map[string]*MsgAttr) ([]byte, error) {
	result := DeviceAttrUpdate{BaseMessage: baseMessage, Attributes: attrs}
	payload, err := json.Marshal(result)
	if err != nil {
		return []byte(""), err
	}
	return payload, nil
}

// MembershipGetResult membership get result
type MembershipGetResult struct {
	BaseMessage
	Devices []Device `json:"devices"`
}

// BuildMembershipGetResult build membership
func BuildMembershipGetResult(baseMessage BaseMessage, devices []*Device) ([]byte, error) {
	result := make([]Device, 0, len(devices))
	for _, v := range devices {
		result = append(result, Device{
			ID:          v.ID,
			Name:        v.Name,
			Description: v.Description,
			State:       v.State,
			LastOnline:  v.LastOnline,
			Attributes:  v.Attributes})
	}
	payload, err := json.Marshal(MembershipGetResult{BaseMessage: baseMessage, Devices: result})
	if err != nil {
		return []byte(""), err
	}
	return payload, nil
}

// DeviceTwinResult device get result
type DeviceTwinResult struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

// BuildDeviceTwinResult build device twin result, 0:get,1:update,2:sync
func BuildDeviceTwinResult(baseMessage BaseMessage, twins map[string]*MsgTwin, dealType int) ([]byte, error) {
	result := make(map[string]*MsgTwin)
	if dealType == 0 {
		for k, v := range twins {
			if v == nil {
				result[k] = nil
				continue
			}
			if v.Metadata != nil && strings.Compare(v.Metadata.Type, dtcommon.TypeDeleted) == 0 {
				continue
			}
			twin := *v

			twin.ActualVersion = nil
			twin.ExpectedVersion = nil
			result[k] = &twin
		}
	} else {
		result = twins
	}

	payload, err := json.Marshal(DeviceTwinResult{BaseMessage: baseMessage, Twin: result})
	if err != nil {
		return []byte(""), err
	}
	return payload, nil
}

// BuildErrorResult build error result
func BuildErrorResult(para Parameter) ([]byte, error) {
	result := Result{BaseMessage: BaseMessage{Timestamp: time.Now().UnixNano() / 1e6,
		EventID: para.EventID},
		Code:   para.Code,
		Reason: para.Reason}
	errorResult, err := json.Marshal(result)
	if err != nil {
		return []byte(""), err
	}
	return errorResult, nil
}

// DeviceUpdate device update
type DeviceUpdate struct {
	BaseMessage
	State      string              `json:"state,omitempty"`
	Attributes map[string]*MsgAttr `json:"attributes"`
}

// UnmarshalDeviceUpdate unmarshal device update
func UnmarshalDeviceUpdate(payload []byte) (*DeviceUpdate, error) {
	var get DeviceUpdate
	err := json.Unmarshal(payload, &get)
	if err != nil {
		return nil, err
	}
	return &get, nil
}

// DeviceTwinDelta devicetwin
type DeviceTwinDelta struct {
	BaseMessage
	Twin  map[string]*MsgTwin `json:"twin"`
	Delta map[string]string   `json:"delta"`
}

// BuildDeviceTwinDelta  build device twin delta
func BuildDeviceTwinDelta(baseMessage BaseMessage, twins map[string]*MsgTwin) ([]byte, bool) {
	result := make(map[string]*MsgTwin, len(twins))
	delta := make(map[string]string)
	for k, v := range twins {
		if v.Metadata != nil && strings.Compare(v.Metadata.Type, dtcommon.TypeDeleted) == 0 {
			continue
		}
		var expectedValue, actualValue string
		if v.Expected != nil && v.Expected.Value != nil {
			expectedValue = *v.Expected.Value
		}
		if expectedValue == "" {
			continue
		}

		if v.Actual != nil && v.Actual.Value != nil {
			actualValue = *v.Actual.Value
		}
		if expectedValue != actualValue {
			delta[k] = expectedValue
		}
		twin := *v

		twin.ActualVersion = nil
		twin.ExpectedVersion = nil

		result[k] = &twin
	}
	payload, err := json.Marshal(DeviceTwinDelta{BaseMessage: baseMessage, Twin: result, Delta: delta})
	if err != nil {
		return []byte(""), false
	}
	if len(delta) > 0 {
		return payload, true
	}
	return payload, false
}

// BuildDeviceTwinDocument  build device twin document
func BuildDeviceTwinDocument(baseMessage BaseMessage, twins map[string]*TwinDoc) ([]byte, bool) {
	payload, err := json.Marshal(DeviceTwinDocument{BaseMessage: baseMessage, Twin: twins})
	if err != nil {
		return []byte(""), false
	}
	return payload, true
}
