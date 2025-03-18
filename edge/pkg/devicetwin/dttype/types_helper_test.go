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

package dttype

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
)

// TestUnmarshalMembershipDetail is function to test UnmarshalMembershipDetails()
func TestUnmarshalMembershipDetail(t *testing.T) {
	assert := assert.New(t)

	var memDetail MembershipDetail
	bytesMemDetail, _ := json.Marshal(memDetail)
	tests := []struct {
		name             string
		membershipDetail []byte
		want             *MembershipDetail
		wantErr          error
	}{
		{
			name:             "UnMarshalMembershipDetailTest-WrongFormat",
			membershipDetail: []byte(""),
			want:             nil,
			wantErr:          errors.New("unexpected end of JSON input"),
		},
		{
			name:             "UnMarshalMembershipDetailTest-CorrectFormat",
			membershipDetail: bytesMemDetail,
			want:             &memDetail,
			wantErr:          nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnmarshalMembershipDetail(test.membershipDetail)
			if err != nil {
				assert.EqualError(err, test.wantErr.Error())
			} else {
				assert.NoError(err)
			}
			assert.Equal(test.want, got)
		})
	}
}

// TestUnmarshalMembershipUpdate is function to test UnmarshalMembershipUpdate().
func TestUnmarshalMembershipUpdate(t *testing.T) {
	assert := assert.New(t)

	var memUpdate MembershipUpdate
	bytesMemUpdate, _ := json.Marshal(memUpdate)
	tests := []struct {
		name             string
		membershipUpdate []byte
		want             *MembershipUpdate
		wantErr          error
	}{
		{
			name:             "UnMarshalMembershipUpdateTest-WrongFormat",
			membershipUpdate: []byte(""),
			want:             nil,
			wantErr:          errors.New("unexpected end of JSON input"),
		},
		{
			name:             "UnMarshalMembershipUpdateTest-RightFormat",
			membershipUpdate: bytesMemUpdate,
			want:             &memUpdate,
			wantErr:          nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnmarshalMembershipUpdate(test.membershipUpdate)
			if err != nil {
				assert.EqualError(err, test.wantErr.Error())
			} else {
				assert.NoError(err)
			}
			assert.Equal(test.want, got)
		})
	}
}

// TestUnmarshalBaseMessage is function to test UnmarshalBaseMessage().
func TestUnmarshalBaseMessage(t *testing.T) {
	assert := assert.New(t)

	var baseMessage BaseMessage
	bytesBaseMessage, _ := json.Marshal(baseMessage)
	tests := []struct {
		name    string
		baseMsg []byte
		want    *BaseMessage
		wantErr error
	}{
		{
			name:    "UnmarshalBaseMessageTest-WrongFormat",
			baseMsg: []byte(""),
			want:    nil,
			wantErr: errors.New("unexpected end of JSON input"),
		},
		{
			name:    "UnmarshalBaseMessageTest-RightFormat",
			baseMsg: bytesBaseMessage,
			want:    &baseMessage,
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnmarshalBaseMessage(test.baseMsg)
			if err != nil {
				assert.EqualError(err, test.wantErr.Error())
			} else {
				assert.NoError(err)
			}
			assert.Equal(test.want, got)
		})
	}
}

// TestDeviceAttrToMsgAttr is function to test DeviceAttrtoMsgAttr().
func TestDeviceAttrToMsgAttr(t *testing.T) {
	assert := assert.New(t)

	var devAttr []dtclient.DeviceAttr
	attr := dtclient.DeviceAttr{
		ID:          00,
		DeviceID:    "DeviceA",
		Name:        "SensorTag",
		Description: "Sensor",
		Value:       "Temperature",
		Optional:    true,
		AttrType:    "float",
		Metadata:    "CelsiusScale",
	}

	devAttr = append(devAttr, attr)
	wantAttr := make(map[string]*MsgAttr)
	wantAttr[attr.Name] = &MsgAttr{
		Value:    attr.Value,
		Optional: &attr.Optional,
		Metadata: &TypeMetadata{
			Type: attr.AttrType,
		},
	}

	tests := []struct {
		name        string
		deviceAttrs []dtclient.DeviceAttr
		want        map[string]*MsgAttr
	}{
		{
			name:        "DeviceAttrToMsgAttr",
			deviceAttrs: devAttr,
			want:        wantAttr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceAttrToMsgAttr(tt.deviceAttrs)
			assert.Equal(len(tt.want), len(got), "DeviceAttrToMsgAttr failed due to wrong map size")
			for gotKey, gotValue := range got {
				wantValue, keyPresent := tt.want[gotKey]
				assert.True(keyPresent, "DeviceAttrToMsgAttr failed due to wrong key %v", gotKey)
				assert.Equal(wantValue.Metadata, gotValue.Metadata, "Error in DeviceAttrToMsgAttr() Got wrong value for key %v", gotKey)
				assert.Equal(wantValue.Optional, gotValue.Optional, "Error in DeviceAttrToMsgAttr() Got wrong value for key %v", gotKey)
				assert.Equal(wantValue.Value, gotValue.Value, "Error in DeviceAttrToMsgAttr() Got wrong value for key %v", gotKey)
			}
		})
	}
}

// createDeviceTwin() is function to create an array of DeviceTwin.
func createDeviceTwin(devTwin dtclient.DeviceTwin) []dtclient.DeviceTwin {
	deviceTwin := []dtclient.DeviceTwin{}
	deviceTwin = append(deviceTwin, devTwin)
	return deviceTwin
}

// createMessageTwinFromDeviceTwin() is function to create MessageTwin corresponding to a particular DeviceTwin.
func createMessageTwinFromDeviceTwin(t *testing.T, devTwin dtclient.DeviceTwin) map[string]*MsgTwin {
	var (
		expectedMeta ValueMetadata
		err          error
	)
	expectedValue := &TwinValue{Value: &devTwin.Expected}
	err = json.Unmarshal([]byte(devTwin.ExpectedMeta), &expectedMeta)
	assert.NoError(t, err)
	expectedValue.Metadata = &expectedMeta
	var actualMeta ValueMetadata
	actualValue := &TwinValue{Value: &devTwin.Actual}
	err = json.Unmarshal([]byte(devTwin.ActualMeta), &actualMeta)
	assert.NoError(t, err)
	actualValue.Metadata = &actualMeta
	var expectedVersion TwinVersion
	err = json.Unmarshal([]byte(devTwin.ExpectedVersion), &expectedVersion)
	assert.NoError(t, err)
	var actualVersion TwinVersion
	err = json.Unmarshal([]byte(devTwin.ActualVersion), &actualVersion)
	assert.NoError(t, err)
	msgTwins := make(map[string]*MsgTwin)
	msgTwin := &MsgTwin{
		Optional: &devTwin.Optional,
		Metadata: &TypeMetadata{
			Type: devTwin.AttrType,
		},
		Expected:        expectedValue,
		Actual:          actualValue,
		ExpectedVersion: &expectedVersion,
		ActualVersion:   &actualVersion,
	}
	msgTwins[devTwin.Name] = msgTwin
	return msgTwins
}

// TestDeviceTwinToMsgTwin is function to test DeviceTwinToMsgTwin().
func TestDeviceTwinToMsgTwin(t *testing.T) {
	assert := assert.New(t)

	devTwin := dtclient.DeviceTwin{
		ID:              00,
		DeviceID:        "DeviceA",
		Name:            "SensorTag",
		Description:     "Sensor",
		Expected:        "ON",
		Actual:          "ON",
		ExpectedVersion: `{"cloud": 10, "Edge": 10}`,
		ActualVersion:   `{"cloud": 10, "Edge": 10}`,
		Optional:        true,
		ExpectedMeta:    fmt.Sprintf(`{"timestamp": %d}`, time.Now().Unix()),
		ActualMeta:      fmt.Sprintf(`{"timestamp": %d}`, time.Now().Unix()),
		AttrType:        "Temperature",
	}
	deviceTwin := createDeviceTwin(devTwin)
	msgTwins := createMessageTwinFromDeviceTwin(t, devTwin)
	tests := []struct {
		name        string
		deviceTwins []dtclient.DeviceTwin
		want        map[string]*MsgTwin
	}{
		{
			name:        "DeviceTwinToMsgTwinTest",
			deviceTwins: deviceTwin,
			want:        msgTwins,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(tt.want, DeviceTwinToMsgTwin(tt.deviceTwins))
		})
	}
}

// TestMsgAttrToDeviceAttr is function to test MsgAttrToDeviceAttr().
func TestMsgAttrToDeviceAttr(t *testing.T) {
	assert := assert.New(t)

	optional := true
	metadata := &TypeMetadata{
		Type: "string",
	}
	msgAttr := MsgAttr{
		Optional: &optional,
		Metadata: metadata,
	}
	wantDeviceAttr := dtclient.DeviceAttr{
		Name:     "Sensor",
		AttrType: metadata.Type,
		Optional: optional,
	}

	tests := []struct {
		name         string
		attrname     string
		msgAttribute *MsgAttr
		want         dtclient.DeviceAttr
	}{
		{
			name:         "MsgAttrToDeviceAttrTest",
			attrname:     "Sensor",
			msgAttribute: &msgAttr,
			want:         wantDeviceAttr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(tt.want, MsgAttrToDeviceAttr(tt.attrname, tt.msgAttribute))
		})
	}
}

// TestCopyMsgTwin is function to test CopyMsgTwin().
func TestCopyMsgTwin(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name      string
		msgTwin   *MsgTwin
		noVersion bool
		want      MsgTwin
	}{
		{
			name: "CopyMsgTwinTest/noVersion-true",
			msgTwin: &MsgTwin{
				ActualVersion: &TwinVersion{
					CloudVersion: 10,
					EdgeVersion:  10,
				},
				ExpectedVersion: &TwinVersion{
					CloudVersion: 11,
					EdgeVersion:  11,
				},
			},
			noVersion: true,
			want: MsgTwin{
				ActualVersion:   nil,
				ExpectedVersion: nil,
			},
		},
		{
			name: "CopyMsgTwinTest/noVersion-false",
			msgTwin: &MsgTwin{
				ActualVersion: &TwinVersion{
					CloudVersion: 10,
					EdgeVersion:  10,
				},
				ExpectedVersion: &TwinVersion{
					CloudVersion: 11,
					EdgeVersion:  11,
				},
			},
			noVersion: false,
			want: MsgTwin{
				ActualVersion: &TwinVersion{
					CloudVersion: 10,
					EdgeVersion:  10,
				},
				ExpectedVersion: &TwinVersion{
					CloudVersion: 11,
					EdgeVersion:  11,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CopyMsgTwin(tt.msgTwin, tt.noVersion)
			assert.Equal(tt.want, got)
		})
	}
}

// TestCopyMsgAttr is function to test CopyMsgAttr().
func TestCopyMsgAttr(t *testing.T) {
	assert := assert.New(t)

	optional := true
	metaData := TypeMetadata{Type: "Attribute"}
	tests := []struct {
		name    string
		msgAttr *MsgAttr
		want    MsgAttr
	}{
		{
			name: "CopyMsgAttrTest",
			msgAttr: &MsgAttr{
				Value:    "value",
				Optional: &optional,
				Metadata: &metaData,
			},
			want: MsgAttr{
				Value:    "value",
				Optional: &optional,
				Metadata: &metaData,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CopyMsgAttr(tt.msgAttr)
			assert.Equal(tt.want, got)
		})
	}
}

// TestMsgTwinToDeviceTwin is function to test MsgTwinToDeviceTwin().
func TestMsgTwinToDeviceTwin(t *testing.T) {
	assert := assert.New(t)

	optional := true
	metadata := TypeMetadata{Type: "Twin"}
	tests := []struct {
		name     string
		twinName string
		msgTwin  *MsgTwin
		want     dtclient.DeviceTwin
	}{
		{
			name:     "MsgTwinToDeviceTwinTest",
			twinName: "DeviceA",
			msgTwin: &MsgTwin{
				Optional: &optional,
				Metadata: &metadata,
			},
			want: dtclient.DeviceTwin{
				Name:     "DeviceA",
				AttrType: metadata.Type,
				Optional: optional,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := MsgTwinToDeviceTwin(test.twinName, test.msgTwin)
			assert.Equal(test.want, got)
		})
	}
}

// TestBuildDeviceState is function to test BuildDeviceCloudMsgState().
func TestBuildDeviceState(t *testing.T) {
	assert := assert.New(t)

	baseMessage := BaseMessage{EventID: uuid.New().String(), Timestamp: time.Now().UnixNano() / 1e6}
	device := Device{
		Name:       "SensorTag",
		State:      "ON",
		LastOnline: "Today",
	}
	deviceCloudMsg := DeviceCloudMsg{
		Name:           "SensorTag",
		State:          "ON",
		LastOnlineTime: "Today",
	}
	deviceMsg := DeviceMsg{
		BaseMessage:    baseMessage,
		DeviceCloudMsg: deviceCloudMsg,
	}
	want, _ := json.Marshal(deviceMsg)
	tests := []struct {
		name        string
		baseMessage BaseMessage
		device      Device
		want        []byte
		wantErr     error
	}{
		{
			name:        "BuildDeviceStateTest",
			baseMessage: baseMessage,
			device:      device,
			want:        want,
			wantErr:     nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildDeviceCloudMsgState(test.baseMessage, test.device)
			assert.Equal(test.wantErr, err)
			assert.Equal(test.want, got)
		})
	}
}

// TestBuildDeviceAttrUpdate is function to test BuildDeviceAttrUpdate().
func TestBuildDeviceAttrUpdate(t *testing.T) {
	assert := assert.New(t)

	baseMessage := BaseMessage{
		EventID:   uuid.New().String(),
		Timestamp: time.Now().UnixNano() / 1e6,
	}
	attr := dtclient.DeviceAttr{
		ID:          00,
		DeviceID:    "DeviceA",
		Name:        "SensorTag",
		Description: "Sensor",
		Value:       "Temperature",
		Optional:    true,
		AttrType:    "float",
		Metadata:    "CelsiusScale",
	}
	attrs := make(map[string]*MsgAttr)
	attrs[attr.Name] = &MsgAttr{
		Value:    attr.Value,
		Optional: &attr.Optional,
		Metadata: &TypeMetadata{
			Type: attr.AttrType,
		},
	}
	devAttrUpdate := DeviceAttrUpdate{
		BaseMessage: baseMessage,
		Attributes:  attrs,
	}
	bytesDevAttrUpdate, _ := json.Marshal(devAttrUpdate)
	tests := []struct {
		name        string
		baseMessage BaseMessage
		attrs       map[string]*MsgAttr
		want        []byte
		wantErr     error
	}{
		{
			name:        "BuildDeviceAttrUpdateTest",
			baseMessage: baseMessage,
			attrs:       attrs,
			wantErr:     nil,
			want:        bytesDevAttrUpdate,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildDeviceAttrUpdate(test.baseMessage, test.attrs)
			assert.Equal(test.wantErr, err)
			assert.Equal(test.want, got)
		})
	}
}

// createMessageAttribute() is function to create a map of message attributes.
func createMessageAttribute() map[string]*MsgAttr {
	attrs := make(map[string]*MsgAttr)
	optional := true
	metadata := &TypeMetadata{
		Type: "string",
	}
	msgAttr := MsgAttr{
		Optional: &optional,
		Metadata: metadata,
	}
	attrs["SensorTag"] = &msgAttr
	return attrs
}

// createDevice is function to create an array of devices.
func createDevice() []*Device {
	attrs := createMessageAttribute()
	devices := []*Device{}
	device := &Device{
		ID:          "id1",
		Name:        "SensorTag",
		Description: "Sensor",
		State:       "ON",
		LastOnline:  "TODAY",
		Attributes:  attrs,
	}
	devices = append(devices, device)
	return devices
}

// createMembershipGetResult is function to create MembershipGetResult with given base message.
func createMembershipGetResult(message BaseMessage) MembershipGetResult {
	attrs := createMessageAttribute()
	var devices []Device
	device := Device{
		ID:          "id1",
		Name:        "SensorTag",
		Description: "Sensor",
		State:       "ON",
		LastOnline:  "TODAY",
		Attributes:  attrs,
	}
	devices = append(devices, device)

	memGetResult := MembershipGetResult{BaseMessage: message, Devices: devices}
	return memGetResult
}

// TestBuildMembershipGetResult is function to test BuildMembershipGetResult().
func TestBuildMembershipGetResult(t *testing.T) {
	assert := assert.New(t)

	baseMessage := BaseMessage{EventID: uuid.New().String(), Timestamp: time.Now().UnixNano() / 1e6}
	devices := createDevice()
	memGetResult := createMembershipGetResult(baseMessage)
	bytesMemGetResult, _ := json.Marshal(memGetResult)
	tests := []struct {
		name        string
		baseMessage BaseMessage
		devices     []*Device
		want        []byte
		wantErr     error
	}{
		{
			name:        "BuildMembershipGetResultTest",
			baseMessage: baseMessage,
			devices:     devices,
			want:        bytesMemGetResult,
			wantErr:     nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildMembershipGetResult(test.baseMessage, test.devices)
			assert.Equal(test.wantErr, err)
			assert.Equal(test.want, got)
		})
	}
}

// createMessageTwin() is function to create a map of MessageTwin with MetaDataType updated and deleted.
func createMessageTwin() map[string]*MsgTwin {
	msgTwins := make(map[string]*MsgTwin)
	msgTwins["empty"] = nil
	msgTwins[dtcommon.TypeDeleted] = generateTwinActualExpected(dtcommon.TypeDeleted, "", "")
	msgTwins["updated"] = generateTwinActualExpected(dtcommon.TypeUpdated, "", "")
	return msgTwins
}

// createDeviceTwinResultDealTypeGet() is function to create DeviceTwinResult with DealType 0(Get).
func createDeviceTwinResultDealTypeGet(baseMessage BaseMessage) DeviceTwinResult {
	resultDealType0Twin := make(map[string]*MsgTwin)
	resultDealType0Twin["empty"] = nil
	resultDealType0Twin["updated"] = generateTwinActualExpected(dtcommon.TypeUpdated, "", "")
	devTwinResult := DeviceTwinResult{
		BaseMessage: baseMessage,
		Twin:        resultDealType0Twin,
	}
	return devTwinResult
}

// createDeviceTwinResult() is function to create DeviceTwinResult with DealType of value other than 0(1-Update, 2-Sync).
func createDeviceTwinResult(baseMessage BaseMessage) DeviceTwinResult {
	resultDealTypeTwin := make(map[string]*MsgTwin)
	msgTwins := createMessageTwin()
	resultDealTypeTwin = msgTwins
	devTwinResult := DeviceTwinResult{
		BaseMessage: baseMessage,
		Twin:        resultDealTypeTwin,
	}
	return devTwinResult
}

// TestBuildDeviceTwinResult is function to test BuildDeviceTwinResult().
func TestBuildDeviceTwinResult(t *testing.T) {
	assert := assert.New(t)

	baseMessage := BaseMessage{EventID: uuid.New().String(), Timestamp: time.Now().UnixNano() / 1e6}
	msgTwins := createMessageTwin()
	devTwinResultDealType0 := createDeviceTwinResultDealTypeGet(baseMessage)
	bytesDealType0, _ := json.Marshal(devTwinResultDealType0)
	devTwinResult1 := createDeviceTwinResult(baseMessage)
	bytesDealType1, _ := json.Marshal(devTwinResult1)
	tests := []struct {
		name        string
		baseMessage BaseMessage
		twins       map[string]*MsgTwin
		dealType    int
		want        []byte
		wantErr     error
	}{
		{
			name:        "Test1",
			baseMessage: baseMessage,
			twins:       msgTwins,
			dealType:    0,
			want:        bytesDealType0,
			wantErr:     nil,
		},
		{
			name:        "Test2",
			baseMessage: baseMessage,
			twins:       msgTwins,
			dealType:    1,
			want:        bytesDealType1,
			wantErr:     nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildDeviceTwinResult(test.baseMessage, test.twins, test.dealType)
			assert.Equal(test.wantErr, err)
			assert.Equal(test.want, got)
		})
	}
}

// TestBuildErrorResult is function to test BuildErrorResult().
func TestBuildErrorResult(t *testing.T) {
	assert := assert.New(t)

	result := Result{BaseMessage: BaseMessage{
		EventID: ""},
		Code:   1,
		Reason: ""}
	tests := []struct {
		name    string
		para    Parameter
		want    Result
		wantErr error
	}{
		{
			name:    "BuildErrorResultTest",
			para:    Parameter{EventID: "", Code: 1, Reason: ""},
			want:    result,
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildErrorResult(test.para)
			var gotResult Result
			err = json.Unmarshal(got, &gotResult)
			assert.NoError(err)
			assert.Equal(test.wantErr, err)
			assert.Equal(test.want.EventID, gotResult.EventID)
			assert.Equal(test.want.Reason, gotResult.Reason)
			assert.Equal(test.want.Code, gotResult.Code)
		})
	}
}

// TestUnmarshalDeviceUpdate is function to test UnmarshalDeviceUpdate().
func TestUnmarshalDeviceUpdate(t *testing.T) {
	assert := assert.New(t)

	var devUpdate DeviceUpdate
	bytesDevUpdate, _ := json.Marshal(devUpdate)
	tests := []struct {
		name    string
		payload []byte
		want    *DeviceUpdate
		wantErr error
	}{
		{
			name:    "UnmarshalDeviceUpdateTest",
			payload: bytesDevUpdate,
			want:    &devUpdate,
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnmarshalDeviceUpdate(test.payload)
			assert.Equal(test.wantErr, err)
			assert.Equal(test.want, got)
		})
	}
}

// createMessageTwinWithDiffValues() is function to create MessageTwin with actual and expected values.
func createMessageTwinWithDiffValues(BaseMessage) map[string]*MsgTwin {
	msgTwins := make(map[string]*MsgTwin)
	expected := "ON"
	actual := "OFF"
	msgTwins[dtcommon.TypeDeleted] = generateTwinActualExpected(dtcommon.TypeDeleted, "", "")
	msgTwins[dtcommon.DeviceTwinModule] = generateTwinActualExpected(dtcommon.TypeUpdated, expected, actual)
	msgTwins["expected"] = generateTwinActualExpected(dtcommon.TypeUpdated, expected, "")
	msgTwins["actual"] = generateTwinActualExpected(dtcommon.TypeUpdated, "", expected)
	return msgTwins
}

// createMessageTwinWithSameValues() is function to create MessageTwin with same actual and expected values.
func createMessageTwinWithSameValues() map[string]*MsgTwin {
	value := "ON"
	msgTwin := make(map[string]*MsgTwin)
	msgTwin["twins"] = generateTwinActualExpected(dtcommon.TypeUpdated, value, value)
	return msgTwin
}

// createMessageTwinAndDeltaWithDiffValues is function to create MessageTwin and Delta with different actual and expected values.
func createMessageTwinAndDeltaWithDiffValues() (map[string]*MsgTwin, map[string]string) {
	delta := make(map[string]string)
	expected := "ON"
	actual := "OFF"
	delta["twin"] = expected
	delta["expected"] = expected
	resultTwin := make(map[string]*MsgTwin)
	resultTwin["twin"] = generateTwinActualExpected(dtcommon.TypeUpdated, expected, actual)
	resultTwin["expected"] = generateTwinActualExpected(dtcommon.TypeUpdated, expected, "")
	return resultTwin, delta
}

// createMessageTwinAndDeltaWithSameValues is function to create MessageTwin and Delta with same actual and expected values.
func createMessageTwinAndDeltaWithSameValues() (map[string]*MsgTwin, map[string]string) {
	value := "ON"
	deltas := make(map[string]string)
	resultTwins := make(map[string]*MsgTwin)
	resultTwins["twins"] = generateTwinActualExpected(dtcommon.TypeUpdated, value, value)
	return resultTwins, deltas
}

// TestBuildDeviceTwinDelta is function to test BuildDeviceTwinDelta().
func TestBuildDeviceTwinDelta(t *testing.T) {
	assert := assert.New(t)

	baseMessage := BaseMessage{EventID: "Event1", Timestamp: time.Now().UnixNano() / 1e6}
	msgTwins := createMessageTwinWithDiffValues(baseMessage)
	delta := make(map[string]string)
	resultTwinDiffValues, delta := createMessageTwinAndDeltaWithDiffValues()
	bytesResultTwinDiffValues, _ := json.Marshal(DeviceTwinDelta{BaseMessage: baseMessage, Twin: resultTwinDiffValues, Delta: delta})
	msgTwin := createMessageTwinWithSameValues()
	resultTwinSameValues, deltas := createMessageTwinAndDeltaWithSameValues()
	bytesResultTwinSameValues, _ := json.Marshal(DeviceTwinDelta{BaseMessage: baseMessage, Twin: resultTwinSameValues, Delta: deltas})
	tests := []struct {
		name        string
		baseMessage BaseMessage
		twins       map[string]*MsgTwin
		want        []byte
		wantBool    bool
	}{
		{
			name:        "BuildDeviceTwinDeltaTest",
			baseMessage: baseMessage,
			twins:       msgTwins,
			want:        bytesResultTwinDiffValues,
			wantBool:    true,
		},
		{
			name:        "BuildDeviceTwinTest",
			baseMessage: baseMessage,
			twins:       msgTwin,
			want:        bytesResultTwinSameValues,
			wantBool:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, gotBool := BuildDeviceTwinDelta(test.baseMessage, test.twins)
			assert.Equal(test.want, got)
			assert.Equal(test.wantBool, gotBool)
		})
	}
}

// TestBuildDeviceTwinDocument is function to test BuildDeviceTwinDocument().
func TestBuildDeviceTwinDocument(t *testing.T) {
	assert := assert.New(t)

	twinDoc := make(map[string]*TwinDoc)
	doc := TwinDoc{
		LastState:    generateTwinActualExpected(dtcommon.TypeUpdated, "", ""),
		CurrentState: generateTwinActualExpected(dtcommon.TypeDeleted, "", ""),
	}
	twinDoc["SensorTag"] = &doc
	timeStamp := time.Now().UnixNano() / 1e6
	devTwinDoc := DeviceTwinDocument{
		BaseMessage: BaseMessage{
			EventID:   "",
			Timestamp: timeStamp,
		},
		Twin: twinDoc,
	}
	bytesdevTwinDoc, _ := json.Marshal(devTwinDoc)
	tests := []struct {
		name        string
		baseMessage BaseMessage
		twins       map[string]*TwinDoc
		want        []byte
		wantBool    bool
	}{
		{
			name:        "BuildDeviceTwinDocumentTest",
			baseMessage: BaseMessage{EventID: "", Timestamp: timeStamp},
			twins:       twinDoc,
			want:        bytesdevTwinDoc,
			wantBool:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, gotBool := BuildDeviceTwinDocument(test.baseMessage, test.twins)
			assert.Equal(test.want, got, "BuildDeviceTwinDocument() got = %v, want %v", got, test.want)
			assert.Equal(test.wantBool, gotBool, "BuildDeviceTwinDocument() gotBool = %v, want %v", gotBool, test.wantBool)
		})
	}
}

func generateTwinActualExpected(t, expected, actual string) *MsgTwin {
	return &MsgTwin{
		Metadata: &TypeMetadata{
			Type: t,
		},
		Expected: &TwinValue{
			Value: &expected,
		},
		Actual: &TwinValue{
			Value: &actual,
		},
	}
}
