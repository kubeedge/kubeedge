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
	"reflect"
	"testing"
	"time"

	"github.com/satori/go.uuid"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
)

// TestUnmarshalMembershipDetail is function to test UnmarshalMembershipDetails()
func TestUnmarshalMembershipDetail(t *testing.T) {
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
				if !reflect.DeepEqual(err.Error(), test.wantErr.Error()) {
					t.Errorf("Error Got = %v,Want =%v", err.Error(), test.wantErr.Error())
					return
				}
			} else {
				if !reflect.DeepEqual(err, test.wantErr) {
					t.Errorf("Error Got = %v,Want =%v", err, test.wantErr)
					return
				}
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalMembershipDetail() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestUnmarshalMembershipUpdate is function to test UnmarshalMembershipUpdate().
func TestUnmarshalMembershipUpdate(t *testing.T) {
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
				if !reflect.DeepEqual(err.Error(), test.wantErr.Error()) {
					t.Errorf("Error Got = %v,Want =%v", err.Error(), test.wantErr.Error())
					return
				}
			} else {
				if !reflect.DeepEqual(err, test.wantErr) {
					t.Errorf("Error Got = %v,Want =%v", err, test.wantErr)
					return
				}
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalMembershipUpdate() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestUnmarshalBaseMessage is function to test UnmarshalBaseMessage().
func TestUnmarshalBaseMessage(t *testing.T) {
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
				if !reflect.DeepEqual(err.Error(), test.wantErr.Error()) {
					t.Errorf("Error Got = %v,Want =%v", err.Error(), test.wantErr.Error())
					return
				}
			} else {
				if !reflect.DeepEqual(err, test.wantErr) {
					t.Errorf("Error Got = %v,Want =%v", err, test.wantErr)
					return
				}
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalBaseMessage() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestDeviceAttrToMsgAttr is function to test DeviceAttrtoMsgAttr().
func TestDeviceAttrToMsgAttr(t *testing.T) {
	var devAttr []dtclient.DeviceAttr
	attr := dtclient.DeviceAttr{ID: 00, DeviceID: "DeviceA", Name: "SensorTag", Description: "Sensor", Value: "Temperature", Optional: true, AttrType: "float", Metadata: "CelsiusScale"}
	devAttr = append(devAttr, attr)
	wantAttr := make(map[string]*MsgAttr)
	wantAttr[attr.Name] = &MsgAttr{Value: attr.Value, Optional: &attr.Optional, Metadata: &TypeMetadata{Type: attr.AttrType}}
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
			if len(got) != len(tt.want) {
				t.Errorf("DeviceAttrToMsgAttr failed due to wrong map size, Got Size = %v Want = %v", len(got), len(tt.want))
			}
			for gotKey, gotValue := range got {
				keyPresent := false
				for wantKey, wantValue := range tt.want {
					if gotKey == wantKey {
						keyPresent = true
						if !reflect.DeepEqual(gotValue.Metadata, wantValue.Metadata) || !reflect.DeepEqual(gotValue.Optional, wantValue.Optional) || !reflect.DeepEqual(gotValue.Value, wantValue.Value) {
							t.Errorf("Error in DeviceAttrToMsgAttr() Got wrong value for key %v", gotKey)
							return
						}
					}
				}
				if keyPresent == false {
					t.Errorf("DeviceAttrToMsgAttr failed() due to wrong key %v", gotKey)
				}
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
func createMessageTwinFromDeviceTwin(devTwin dtclient.DeviceTwin) map[string]*MsgTwin {
	var expectedMeta ValueMetadata
	expectedValue := &TwinValue{Value: &devTwin.Expected}
	json.Unmarshal([]byte(devTwin.ExpectedMeta), &expectedMeta)
	expectedValue.Metadata = &expectedMeta
	var actualMeta ValueMetadata
	actualValue := &TwinValue{Value: &devTwin.Actual}
	json.Unmarshal([]byte(devTwin.ActualMeta), &actualMeta)
	var expectedVersion TwinVersion
	json.Unmarshal([]byte(devTwin.ExpectedVersion), &expectedVersion)
	var actualVersion TwinVersion
	json.Unmarshal([]byte(devTwin.ActualVersion), &actualVersion)
	msgTwins := make(map[string]*MsgTwin)
	msgTwin := &MsgTwin{Optional: &devTwin.Optional, Metadata: &TypeMetadata{Type: devTwin.AttrType}, Expected: expectedValue, Actual: actualValue, ExpectedVersion: &expectedVersion, ActualVersion: &actualVersion}
	msgTwins[devTwin.Name] = msgTwin
	return msgTwins
}

// TestDeviceTwinToMsgTwin is function to test DeviceTwinToMsgTwin().
func TestDeviceTwinToMsgTwin(t *testing.T) {
	devTwin := dtclient.DeviceTwin{ID: 00, DeviceID: "DeviceA", Name: "SensorTag", Description: "Sensor", Expected: "ON", Actual: "ON", ExpectedVersion: "Version1", ActualVersion: "Version1", Optional: true, ExpectedMeta: "Updation", ActualMeta: "Updation", AttrType: "Temperature"}
	deviceTwin := createDeviceTwin(devTwin)
	msgTwins := createMessageTwinFromDeviceTwin(devTwin)
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
			if got := DeviceTwinToMsgTwin(tt.deviceTwins); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeviceTwinToMsgTwin() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMsgAttrToDeviceAttr is function to test MsgAttrToDeviceAttr().
func TestMsgAttrToDeviceAttr(t *testing.T) {
	optional := true
	metadata := &TypeMetadata{Type: "string"}
	msgAttr := MsgAttr{Optional: &optional, Metadata: metadata}
	wantDeviceAttr := dtclient.DeviceAttr{Name: "Sensor", AttrType: metadata.Type, Optional: optional}
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
			if got := MsgAttrToDeviceAttr(tt.attrname, tt.msgAttribute); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MsgAttrToDeviceAttr() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCopyMsgTwin is function to test CopyMsgTwin().
func TestCopyMsgTwin(t *testing.T) {
	tests := []struct {
		name      string
		msgTwin   *MsgTwin
		noVersion bool
		want      MsgTwin
	}{
		{
			name: "CopyMsgTwinTest/noVersion-true",
			msgTwin: &MsgTwin{
				ActualVersion:   &TwinVersion{CloudVersion: 10, EdgeVersion: 10},
				ExpectedVersion: &TwinVersion{CloudVersion: 11, EdgeVersion: 11},
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
				ActualVersion:   &TwinVersion{CloudVersion: 10, EdgeVersion: 10},
				ExpectedVersion: &TwinVersion{CloudVersion: 11, EdgeVersion: 11},
			},
			noVersion: false,
			want: MsgTwin{
				ActualVersion:   &TwinVersion{CloudVersion: 10, EdgeVersion: 10},
				ExpectedVersion: &TwinVersion{CloudVersion: 11, EdgeVersion: 11},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyMsgTwin(tt.msgTwin, tt.noVersion); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyMsgTwin() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCopyMsgAttr is function to test CopyMsgAttr().
func TestCopyMsgAttr(t *testing.T) {
	optional := true
	metaData := TypeMetadata{Type: "Attribute"}
	tests := []struct {
		name    string
		msgAttr *MsgAttr
		want    MsgAttr
	}{
		{
			name:    "CopyMsgAttrTest",
			msgAttr: &MsgAttr{Value: "value", Optional: &optional, Metadata: &metaData},
			want:    MsgAttr{Value: "value", Optional: &optional, Metadata: &metaData},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyMsgAttr(tt.msgAttr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyMsgAttr() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMsgTwinToDeviceTwin is function to test MsgTwinToDeviceTwin().
func TestMsgTwinToDeviceTwin(t *testing.T) {
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
			if got := MsgTwinToDeviceTwin(test.twinName, test.msgTwin); !reflect.DeepEqual(got, test.want) {
				t.Errorf("MsgTwinToDeviceTwin() = %v, want %v", got, test.want)
			}
		})
	}
}

//TestBuildDeviceState is function to test BuildDeviceState().
func TestBuildDeviceState(t *testing.T) {
	baseMessage := BaseMessage{EventID: uuid.NewV4().String(), Timestamp: time.Now().UnixNano() / 1e6}
	device := Device{Name: "SensorTag", State: "ON", LastOnline: "Today"}
	deviceMsg := DeviceMsg{BaseMessage: baseMessage, Device: device}
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
			got, err := BuildDeviceState(test.baseMessage, test.device)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("Error Got = %v,Want =%v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("BuildDeviceState() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestBuildDeviceAttrUpdate is function to test BuildDeviceAttrUpdate().
func TestBuildDeviceAttrUpdate(t *testing.T) {
	baseMessage := BaseMessage{EventID: uuid.NewV4().String(), Timestamp: time.Now().UnixNano() / 1e6}
	attr := dtclient.DeviceAttr{ID: 00, DeviceID: "DeviceA", Name: "SensorTag", Description: "Sensor", Value: "Temperature", Optional: true, AttrType: "float", Metadata: "CelsiusScale"}
	attrs := make(map[string]*MsgAttr)
	attrs[attr.Name] = &MsgAttr{Value: attr.Value, Optional: &attr.Optional, Metadata: &TypeMetadata{Type: attr.AttrType}}
	devAttrUpdate := DeviceAttrUpdate{BaseMessage: baseMessage, Attributes: attrs}
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
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("Error Got = %v,Want =%v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("BuildDeviceAttrUpdate() = %v, want %v", got, test.want)
			}
		})
	}
}

// createMessageAttribute() is function to create a map of message attributes.
func createMessageAttribute() map[string]*MsgAttr {
	attrs := make(map[string]*MsgAttr)
	optional := true
	metadata := &TypeMetadata{Type: "string"}
	msgAttr := MsgAttr{Optional: &optional, Metadata: metadata}
	attrs["SensorTag"] = &msgAttr
	return attrs
}

// createDevice is function to create an array of devices.
func createDevice() []*Device {
	attrs := createMessageAttribute()
	devices := []*Device{}
	device := &Device{ID: "id1", Name: "SensorTag", Description: "Sensor", State: "ON", LastOnline: "TODAY", Attributes: attrs}
	devices = append(devices, device)
	return devices
}

// createMembershipGetResult is function to create MembershipGetResult with given base message.
func createMembershipGetResult(message BaseMessage) MembershipGetResult {
	attrs := createMessageAttribute()
	devices := []*Device{}
	device := &Device{ID: "id1", Name: "SensorTag", Description: "Sensor", State: "ON", LastOnline: "TODAY", Attributes: attrs}
	devices = append(devices, device)
	wantDevice := []Device{}
	wantDev := Device{ID: device.ID, Name: device.Name, Description: device.Description, State: device.State, LastOnline: device.LastOnline, Attributes: device.Attributes}
	wantDevice = append(wantDevice, wantDev)
	memGetResult := MembershipGetResult{BaseMessage: message, Devices: wantDevice}
	return memGetResult
}

// TestBuildMembershipGetResult is function to test BuildMembershipGetResult().
func TestBuildMembershipGetResult(t *testing.T) {
	baseMessage := BaseMessage{EventID: uuid.NewV4().String(), Timestamp: time.Now().UnixNano() / 1e6}
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
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("BuildMembershipGetResult() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("BuildMembershipGetResult() = %v, want %v", got, test.want)
			}
		})
	}
}

//createMessageTwin() is function to create a map of MessageTwin with MetaDataType updated and deleted.
func createMessageTwin() map[string]*MsgTwin {
	msgTwins := make(map[string]*MsgTwin)
	twinMetadataDeleted := MsgTwin{Metadata: &TypeMetadata{Type: "deleted"}}
	twinMetadataUpdated := MsgTwin{Metadata: &TypeMetadata{Type: "updated"}}
	msgTwins["empty"] = nil
	msgTwins["deleted"] = &twinMetadataDeleted
	msgTwins["updated"] = &twinMetadataUpdated
	return msgTwins
}

// createDeviceTwinResultDealTypeGet() is function to create DeviceTwinResult with DealType 0(Get).
func createDeviceTwinResultDealTypeGet(baseMessage BaseMessage) DeviceTwinResult {
	resultDealType0Twin := make(map[string]*MsgTwin)
	resultDealType0Twin["empty"] = nil
	twinMetadataUpdated := MsgTwin{Metadata: &TypeMetadata{Type: "updated"}}
	resultDealType0Twin["updated"] = &twinMetadataUpdated
	devTwinResult := DeviceTwinResult{BaseMessage: baseMessage, Twin: resultDealType0Twin}
	return devTwinResult
}

// createDeviceTwinResult() is function to create DeviceTwinResult with DealType of value other than 0(1-Update, 2-Sync).
func createDeviceTwinResult(baseMessage BaseMessage) DeviceTwinResult {
	resultDealTypeTwin := make(map[string]*MsgTwin)
	msgTwins := createMessageTwin()
	resultDealTypeTwin = msgTwins
	devTwinResult := DeviceTwinResult{BaseMessage: baseMessage, Twin: resultDealTypeTwin}
	return devTwinResult
}

// TestBuildDeviceTwinResult is function to test BuildDeviceTwinResult().
func TestBuildDeviceTwinResult(t *testing.T) {
	baseMessage := BaseMessage{EventID: uuid.NewV4().String(), Timestamp: time.Now().UnixNano() / 1e6}
	msgTwins := createMessageTwin()
	devTwinResultDealType0 := createDeviceTwinResultDealTypeGet(baseMessage)
	bytesDealType0, _ := json.Marshal(devTwinResultDealType0)
	devTwinResult1 := createDeviceTwinResult(baseMessage)
	bytes_DealType1, _ := json.Marshal(devTwinResult1)
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
			want:        bytes_DealType1,
			wantErr:     nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildDeviceTwinResult(test.baseMessage, test.twins, test.dealType)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("BuildDeviceTwinResult() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("BuildDeviceTwinResult() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestBuildErrorResult is function to test BuildErrorResult().
func TestBuildErrorResult(t *testing.T) {
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
			gotResult := Result{}
			json.Unmarshal(got, &gotResult)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("BuildErrorResult() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult.EventID, test.want.EventID) {
				t.Errorf("BuildErrorResult() error EventID = %v, want = %v", gotResult.EventID, test.want.EventID)
				return
			}
			if !reflect.DeepEqual(gotResult.Reason, test.want.Reason) {
				t.Errorf("BuildErrorResult() error Reason = %v, want = %v", gotResult.Reason, test.want.Reason)
				return
			}
			if !reflect.DeepEqual(gotResult.Code, test.want.Code) {
				t.Errorf("BuildErrorResult() error Code = %v, want = %v", gotResult.Code, test.want.Code)
			}
		})
	}
}

// TestUnmarshalDeviceUpdate is function to test UnmarshalDeviceUpdate().
func TestUnmarshalDeviceUpdate(t *testing.T) {
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
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("UnmarshalDeviceUpdate() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalDeviceUpdate() = %v, want %v", got, test.want)
			}
		})
	}
}

// createMessageTwinWithDiffValues() is function to create MessageTwin with actual and expected values.
func createMessageTwinWithDiffValues(baseMessage BaseMessage) map[string]*MsgTwin {
	msgTwins := make(map[string]*MsgTwin)
	twin_MetadataDeleted := MsgTwin{Metadata: &TypeMetadata{Type: "deleted"}}
	expected := "ON"
	actual := "OFF"
	twin_ActualExpected := MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Expected: &TwinValue{Value: &expected}, Actual: &TwinValue{Value: &actual}}
	msgTwins["deleted"] = &twin_MetadataDeleted
	msgTwins["twin"] = &twin_ActualExpected
	twin_Expected := MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Expected: &TwinValue{Value: &expected}}
	msgTwins["expected"] = &twin_Expected
	twin_Actual := MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Actual: &TwinValue{Value: &expected}}
	msgTwins["actual"] = &twin_Actual
	return msgTwins
}

// createMessageTwinWithSameValues() is function to create MessageTwin with same actual and expected values.
func createMessageTwinWithSameValues() map[string]*MsgTwin {
	value := "ON"
	msgTwin := make(map[string]*MsgTwin)
	twins := MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Actual: &TwinValue{Value: &value}, Expected: &TwinValue{Value: &value}}
	msgTwin["twins"] = &twins
	return msgTwin
}

// createMessageTwinAndDeltaWithDiffValues is function to create MessageTwin and Delta with different actual and expected values.
func createMessageTwinAndDeltaWithDiffValues() (map[string]*MsgTwin, map[string]string) {
	delta := make(map[string]string)
	expected := "ON"
	actual := "OFF"
	twinActualExpected := MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Expected: &TwinValue{Value: &expected}, Actual: &TwinValue{Value: &actual}}
	twinExpected := MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Expected: &TwinValue{Value: &expected}}
	delta["twin"] = *twinActualExpected.Expected.Value
	delta["expected"] = *twinExpected.Expected.Value
	resultTwin := make(map[string]*MsgTwin)
	resultTwin["twin"] = &MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Expected: &TwinValue{Value: &expected}, Actual: &TwinValue{Value: &actual}, ActualVersion: nil, ExpectedVersion: nil}
	resultTwin["expected"] = &MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Expected: &TwinValue{Value: &expected}, ActualVersion: nil, ExpectedVersion: nil}
	return resultTwin, delta
}

// createMessageTwinAndDeltaWithSameValues is function to create MessageTwin and Delta with same actual and expected values.
func createMessageTwinAndDeltaWithSameValues() (map[string]*MsgTwin, map[string]string) {
	value := "ON"
	deltas := make(map[string]string)
	resultTwins := make(map[string]*MsgTwin)
	resultTwins["twins"] = &MsgTwin{Metadata: &TypeMetadata{Type: "updated"}, Actual: &TwinValue{Value: &value}, Expected: &TwinValue{Value: &value}, ActualVersion: nil, ExpectedVersion: nil}
	return resultTwins, deltas
}

// TestBuildDeviceTwinDelta is function to test BuildDeviceTwinDelta().
func TestBuildDeviceTwinDelta(t *testing.T) {
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
			got, got1 := BuildDeviceTwinDelta(test.baseMessage, test.twins)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("BuildDeviceTwinDelta() got = %v, want %v", got, test.want)
			}
			if got1 != test.wantBool {
				t.Errorf("BuildDeviceTwinDelta() got1 = %v, want %v", got1, test.wantBool)
			}
		})
	}
}

// TestBuildDeviceTwinDocument is function to test BuildDeviceTwinDocument().
func TestBuildDeviceTwinDocument(t *testing.T) {
	twinDoc := make(map[string]*TwinDoc)
	doc := TwinDoc{LastState: &MsgTwin{Metadata: &TypeMetadata{"updated"}}, CurrentState: &MsgTwin{Metadata: &TypeMetadata{"deleted"}}}
	twinDoc["SensorTag"] = &doc
	timeStamp := time.Now().UnixNano() / 1e6
	devTwinDoc := DeviceTwinDocument{BaseMessage: BaseMessage{EventID: "", Timestamp: timeStamp}, Twin: twinDoc}
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
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("BuildDeviceTwinDocument() got = %v, want %v", got, test.want)
			}
			if gotBool != test.wantBool {
				t.Errorf("BuildDeviceTwinDocument() gotBool = %v, want %v", gotBool, test.wantBool)
			}
		})
	}
}
