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
)

// TestSetEventID is function to test SetEventID().
func TestSetEventID(t *testing.T) {
	tests := []struct {
		name    string
		eventID string
	}{
		{
			name:    "SetEventIDTest",
			eventID: uuid.NewV4().String(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bs := BaseMessage{}
			bs.SetEventID(test.eventID)
			if bs.EventID != test.eventID {
				t.Errorf("Got wrong EventID, Got = %v, Want = %v", bs.EventID, test.eventID)
			}
		})
	}
}

// TestBuildBaseMessage is function to test BuildBaseMessage().
func TestBuildBaseMessage(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "BuildBaseMessageTest",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := BuildBaseMessage()
			if got.EventID == "" {
				t.Errorf("BuildBaseMessage() failed,Failed to generate EventID")
			}
			if got.Timestamp == 0 {
				t.Errorf("BuildBaseMessage() failed,Failed to get timestamp")
			}
		})
	}
}

// createDevTwinMetaDeleted() is function to create a devicetwin with Metadata "deleted" and add it to map with given deviceID.
func createDevTwinMetaDeleted(deviceID string) map[string]*MsgTwin {
	value := "ON"
	devTwin := map[string]*MsgTwin{}
	metaData := TypeMetadata{Type: "deleted"}
	msgTwin := MsgTwin{Expected: &TwinValue{Value: &value}, Metadata: &metaData}
	devTwin[deviceID] = &msgTwin
	return devTwin
}

// createMembershipUpdateAndBaseMessage() is function to create a base message and membershipupdate.
func createMembershipUpdateAndBaseMessage() ([]byte, BaseMessage) {
	addDevices := []Device{}
	twinAddDevice := make(map[string]*MsgTwin)
	twinAddDevice["DeviceA"] = nil
	deviceA := Device{ID: "DeviceA", Name: "SensorTag", Description: "Sensor", State: "ON", Twin: twinAddDevice}
	addDevices = append(addDevices, deviceA)
	removeDevices := []Device{}
	twinRemDevice := make(map[string]*MsgTwin)
	twinRemDevice["DeviceB"] = nil
	deviceB := Device{ID: "DeviceB", Name: "SensorTag", Description: "Sensor", State: "ON", Twin: twinRemDevice}
	removeDevices = append(removeDevices, deviceB)
	baseMessage := BuildBaseMessage()
	want := MembershipUpdate{
		BaseMessage:   baseMessage,
		AddDevices:    addDevices,
		RemoveDevices: removeDevices,
	}
	wantMembershipUpdate, _ := json.Marshal(want)
	return wantMembershipUpdate, baseMessage
}

// TestMarshalMembershipUpdate is function to test MarshalMembershipUpdate().
func TestMarshalMembershipUpdate(t *testing.T) {
	// Creating membershipupdate and base message
	wantMembershipUpdate, baseMessage := createMembershipUpdateAndBaseMessage()
	tests := []struct {
		name    string
		result  MembershipUpdate
		want    []byte
		wantErr error
	}{
		{
			name:    "MarshalMembershipUpdateTest",
			result:  MembershipUpdate{BaseMessage: baseMessage, AddDevices: []Device{{ID: "DeviceA", Name: "SensorTag", Description: "Sensor", State: "ON", Twin: createDevTwinMetaDeleted("DeviceA")}}, RemoveDevices: []Device{{ID: "DeviceB", Name: "SensorTag", Description: "Sensor", State: "ON", Twin: createDevTwinMetaDeleted("DeviceB")}}},
			wantErr: nil,
			want:    wantMembershipUpdate,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := MarshalMembershipUpdate(test.result)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("MarshalMembershipUpdate() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("MarshalMembershipUpdate() got= %v, want %v", got, test.want)
			}
		})
	}
}

// TestUpdateCloudVersionIncrement is function to test UpdateCloudVersion().
func TestUpdateCloudVersionIncrement(t *testing.T) {
	tests := []struct {
		name         string
		cloudVersion int64
	}{
		{
			name:         "UpdateCloudVersionTest",
			cloudVersion: 10,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tv := TwinVersion{
				CloudVersion: test.cloudVersion,
			}
			tv.UpdateCloudVersion()
			if tv.CloudVersion != test.cloudVersion+1 {
				t.Errorf("UpdateCloudVersion() failed, Got= %v Want = %v", tv.CloudVersion, test.cloudVersion+1)
			}
		})
	}
}

// TestUpdateEdgeVersionIncrement is function to test UpdateEdgeVersion().
func TestUpdateEdgeVersionIncrement(t *testing.T) {
	tests := []struct {
		name        string
		edgeVersion int64
	}{
		{
			name:        "UpdateEdgeVersionTest",
			edgeVersion: 10,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tv := TwinVersion{
				EdgeVersion: test.edgeVersion,
			}
			tv.UpdateEdgeVersion()
			if tv.EdgeVersion != test.edgeVersion+1 {
				t.Errorf("UpdateEdgeVersion() failed, Got= %v Want = %v", tv.EdgeVersion, test.edgeVersion+1)
			}
		})
	}
}

// TestCompareWithCloud is function to test CompareWithCloud().
func TestCompareWithCloud(t *testing.T) {
	tests := []struct {
		name         string
		cloudVersion int64
		edgeVersion  int64
		tvCloud      TwinVersion
		want         bool
	}{
		{
			// Failure Case
			name:         "CompareWithCloudTest-CloudNotUpdated",
			cloudVersion: 10,
			edgeVersion:  10,
			tvCloud: TwinVersion{
				CloudVersion: 11,
				EdgeVersion:  9,
			},
			want: false,
		},
		{
			// Success Case
			name:         "CompareWithCloudTest-CloudUpdated",
			cloudVersion: 10,
			edgeVersion:  10,
			tvCloud: TwinVersion{
				CloudVersion: 11,
				EdgeVersion:  11,
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tv := TwinVersion{
				CloudVersion: test.cloudVersion,
				EdgeVersion:  test.edgeVersion,
			}
			if got := tv.CompareWithCloud(test.tvCloud); got != test.want {
				t.Errorf("TwinVersion.CompareWithCloud() = %v, want %v", got, test.want)
			}
		})
	}
}

//TestUpdateCloudVersion tests UpdateCloudVersion().
func TestUpdateCloudVersion(t *testing.T) {
	twinVersion := TwinVersion{CloudVersion: 1}
	bytesTwinVersion, _ := json.Marshal(twinVersion)
	expectedTwinVersion := TwinVersion{CloudVersion: 2}
	bytesExpectedTwinVersion, _ := json.Marshal(expectedTwinVersion)
	tests := []struct {
		name    string
		version string
		want    string
		wantErr error
	}{
		{
			// Failure Case - wrong input for unmarshal
			name:    "UpdateCloudTest-WrongInput",
			version: "",
			want:    "",
			wantErr: errors.New("unexpected end of JSON input"),
		},
		{
			//Success Case - correct input for unmarshal
			name:    "UpdateCloudTest-CorrectInput",
			version: string(bytesTwinVersion),
			want:    string(bytesExpectedTwinVersion),
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UpdateCloudVersion(test.version)
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
			if got != test.want {
				t.Errorf("UpdateCloudVersion() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestUpdateEdgeVersion test UpdateEdgeVersion().
func TestUpdateEdgeVersion(t *testing.T) {
	twinVersion := TwinVersion{EdgeVersion: 1}
	bytesTwinVersion, _ := json.Marshal(twinVersion)
	expectedTwinVersion := TwinVersion{EdgeVersion: 2}
	bytesExpectedTwinVersion, _ := json.Marshal(expectedTwinVersion)
	tests := []struct {
		name    string
		version string
		want    string
		wantErr error
	}{
		{
			// Failure Case - wrong input for unmarshal
			name:    "UpdateEdgeVersion-UnMarshalFailure",
			version: "",
			want:    "",
			wantErr: errors.New("unexpected end of JSON input"),
		},
		{
			// Success Case - correct input for unmarshal
			name:    "UpdateEdgeVersion-UnMarshalSuccess",
			version: string(bytesTwinVersion),
			want:    string(bytesExpectedTwinVersion),
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UpdateEdgeVersion(test.version)
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
			if got != test.want {
				t.Errorf("UpdateEdgeVersion() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestCompareVersion test CompareVersion().
func TestCompareVersion(t *testing.T) {
	twinCloudUpdated := TwinVersion{
		CloudVersion: 1,
		EdgeVersion:  2,
	}
	bytesTwinCloudUpdated, _ := json.Marshal(twinCloudUpdated)
	twinCloudVersion := TwinVersion{
		CloudVersion: 1,
		EdgeVersion:  0,
	}
	bytesTwinCloudVersion, _ := json.Marshal(twinCloudVersion)
	twinEdgeVersion := TwinVersion{
		CloudVersion: 1,
		EdgeVersion:  1,
	}
	bytesTwinEdgeVersion, _ := json.Marshal(twinEdgeVersion)
	tests := []struct {
		name         string
		cloudversion string
		edgeversion  string
		want         bool
	}{
		{
			// Failure Case - wrong input for unmarshal
			name:         "CompareVersionTest-WrongCloudEdgeVersions",
			cloudversion: "cloudversion",
			edgeversion:  "edgeversion",
			want:         false,
		},
		{
			// Failure Case - wrong input for edgeversion
			name:         "CompareVersionTest-WrongEdgeVersion",
			cloudversion: string(bytesTwinCloudVersion),
			edgeversion:  "edgeversion",
			want:         false,
		},
		{
			// Failure Case - cloud not updated
			name:         "CompareVersionTest-CloudNotUpdated",
			cloudversion: string(bytesTwinCloudVersion),
			edgeversion:  string(bytesTwinEdgeVersion),
			want:         false,
		},
		{
			//Success Case - cloud updated
			name:         "CompareVersionTest-CloudUpdated",
			cloudversion: string(bytesTwinCloudUpdated),
			edgeversion:  string(bytesTwinEdgeVersion),
			want:         true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CompareVersion(test.cloudversion, test.edgeversion); got != test.want {
				t.Errorf("CompareVersion() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestUnmarshalConnectedInfo is function to test UnmarshalConnectedInfo().
func TestUnmarshalConnectedInfo(t *testing.T) {
	connected := ConnectedInfo{
		EventType: "Event",
		TimeStamp: time.Now().UnixNano() / 1e6,
	}
	bytesConnected, _ := json.Marshal(connected)
	tests := []struct {
		name     string
		argument []byte
		want     ConnectedInfo
		wantErr  error
	}{
		{
			// Success Case
			name:     "UnmarshalConnectedInfoTest-CorrectInput",
			argument: bytesConnected,
			want:     connected,
			wantErr:  nil,
		},
		{
			// Failure Case
			name:     "UnmarshalConnectedInfoTest-WrongInput",
			argument: []byte(""),
			want:     ConnectedInfo{},
			wantErr:  errors.New("unexpected end of JSON input"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnmarshalConnectedInfo(test.argument)
			if err != nil && err.Error() != test.wantErr.Error() {
				t.Errorf("UnmarshalConnectedInfo() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalConnectedInfo() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestUnmarshalDeviceTwinDocument is function to test UnmarshalDeviceTwinDocument().
func TestUnmarshalDeviceTwinDocument(t *testing.T) {
	var devTwinDoc DeviceTwinDocument
	byteDevTwinDoc, _ := json.Marshal(devTwinDoc)
	tests := []struct {
		name     string
		argument []byte
		want     *DeviceTwinDocument
		wantErr  error
	}{
		{
			// Success Case
			name:     "UnmarshalDeviceTwinDocumentTest-CorrectInput",
			argument: byteDevTwinDoc,
			wantErr:  nil,
			want:     &devTwinDoc,
		},
		{
			// Failure Case
			name:     "UnmarshalDeviceTwinDocumentTest-WrongInput",
			argument: []byte(""),
			want:     &DeviceTwinDocument{},
			wantErr:  errors.New("unexpected end of JSON input"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnmarshalDeviceTwinDocument(test.argument)
			if err != nil && err.Error() != test.wantErr.Error() {
				t.Errorf("UnmarshalDeviceTwinDocument() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalDeviceTwinDocument() = %v, want %v", got, test.want)
			}
		})
	}
}

// createEmptyDeviceTwinUpdate() is function to create an empty twin update variable.
func createEmptyDeviceTwinUpdate() []byte {
	var emptyTwinUpdate DeviceTwinUpdate
	bytesTwin, _ := json.Marshal(emptyTwinUpdate)
	return bytesTwin
}

// createTwinUpdateWrongKey() is function to create a DeviceTwinUpdate with wrong key.
func createTwinUpdateWrongKey() (DeviceTwinUpdate, []byte) {
	var keyErrorTwinUpdate DeviceTwinUpdate
	twin := make(map[string]*MsgTwin)
	twin["key~"] = &MsgTwin{}
	keyErrorTwinUpdate.Twin = twin
	bytesTwinKeyError, _ := json.Marshal(keyErrorTwinUpdate)
	return keyErrorTwinUpdate, bytesTwinKeyError
}

// createTwinUpdate() is function to create DeviceTwinUpdate with correct actual and expected values.
func createTwinUpdate() (DeviceTwinUpdate, []byte) {
	var keyTwinUpdate DeviceTwinUpdate
	twinKey := make(map[string]*MsgTwin)
	var expected TwinValue
	var actual TwinValue
	value := "value"
	valueMetaData := &ValueMetadata{Timestamp: time.Now().UnixNano() / 1e6}
	expected.Value = &value
	expected.Metadata = valueMetaData
	actual.Value = &value
	actual.Metadata = valueMetaData
	twinKey["key1"] = &MsgTwin{Expected: &expected, Actual: &actual}
	keyTwinUpdate.Twin = twinKey
	bytesTwinKey, _ := json.Marshal(keyTwinUpdate)
	return keyTwinUpdate, bytesTwinKey
}

// createTwinUpdateWrongActual() is function to create  DeviceTwinUpdate having right key with wrong actual value.
func createTwinUpdateWrongActual() (DeviceTwinUpdate, []byte) {
	var keyTwinUpdateActualValueError DeviceTwinUpdate
	twinKeyActualValueError := make(map[string]*MsgTwin)
	var actualValueErrorExpected TwinValue
	var actualValueErrorActual TwinValue
	valueExpected := "value"
	valueActual := "value~"
	valueMetaDataActualValueError := &ValueMetadata{Timestamp: time.Now().UnixNano() / 1e6}
	actualValueErrorExpected.Value = &valueExpected
	actualValueErrorExpected.Metadata = valueMetaDataActualValueError
	actualValueErrorActual.Value = &valueActual
	actualValueErrorActual.Metadata = valueMetaDataActualValueError
	twinKeyActualValueError["key1"] = &MsgTwin{Expected: &actualValueErrorExpected, Actual: &actualValueErrorActual}
	keyTwinUpdateActualValueError.Twin = twinKeyActualValueError
	bytesTwinKeyActualValueError, _ := json.Marshal(keyTwinUpdateActualValueError)
	return keyTwinUpdateActualValueError, bytesTwinKeyActualValueError
}

// createTwinUpdateWrongExpected() is function to create DeviceTwinUpdate having right key with wrong expected value.
func createTwinUpdateWrongExpected() (DeviceTwinUpdate, []byte) {
	var keyTwinUpdateExpectedValueError DeviceTwinUpdate
	twinKeyExpectedValueError := make(map[string]*MsgTwin)
	var expectedValueErrorExpected TwinValue
	var expectedValueErrorActual TwinValue
	valueExpectedValueError := "value~"
	valueMetaDataExpectedValueError := &ValueMetadata{Timestamp: time.Now().UnixNano() / 1e6}
	expectedValueErrorExpected.Value = &valueExpectedValueError
	expectedValueErrorExpected.Metadata = valueMetaDataExpectedValueError
	expectedValueErrorActual.Value = &valueExpectedValueError
	expectedValueErrorActual.Metadata = valueMetaDataExpectedValueError
	twinKeyExpectedValueError["key1"] = &MsgTwin{Expected: &expectedValueErrorExpected, Actual: &expectedValueErrorActual}
	keyTwinUpdateExpectedValueError.Twin = twinKeyExpectedValueError
	bytesTwinKeyExpectedValueError, _ := json.Marshal(keyTwinUpdateExpectedValueError)
	return keyTwinUpdateExpectedValueError, bytesTwinKeyExpectedValueError
}

// TestUnmarshalDeviceTwinUpdate is function to test UnmarshalDeviceTwinUpdate().
func TestUnmarshalDeviceTwinUpdate(t *testing.T) {
	// Creating empty DeviceTwinUpdate variable.
	bytesEmptyTwin := createEmptyDeviceTwinUpdate()
	// Creating DeviceTwinUpdate variable with wrong key entry.
	keyErrorTwinUpdate, bytesTwinKeyError := createTwinUpdateWrongKey()
	// Creating DeviceTwinUpdate variable having right key entry with correct actual and expected values.
	keyTwinUpdate, bytesTwinKey := createTwinUpdate()
	// Creating DeviceTwinUpdate variable having right key entry with wrong expected value.
	keyTwinUpdateExpectedValueError, bytesTwinKeyExpectedValueError := createTwinUpdateWrongExpected()
	// Creating DeviceTwinUpdate variable having right key with wrong actual value
	keyTwinUpdateActualValueError, bytesTwinKeyActualValueError := createTwinUpdateWrongActual()
	tests := []struct {
		name    string
		payload []byte
		want    *DeviceTwinUpdate
		wantErr error
	}{
		{
			// Failure Case - wrong input
			name:    "UnmarshalDeviceTwinUpdateTest-WrongInput",
			payload: []byte(""),
			want:    &DeviceTwinUpdate{},
			wantErr: errors.New("Unmarshal update request body failed, Please check the request"),
		},
		{
			// Failure Case - correct input with empty twin
			name:    "UnmarshalDeviceTwinUpdateTest-EmptyTwin",
			payload: bytesEmptyTwin,
			want:    &DeviceTwinUpdate{},
			wantErr: errors.New("Update twin error, the update request body not have key:twin"),
		},
		{
			// Failure Case - wrong key format
			name:    "UnmarshalDeviceTwinUpdateTest-WrongKeyFormat",
			payload: bytesTwinKeyError,
			want:    &keyErrorTwinUpdate,
			wantErr: errors.New("The key of twin must be only include upper or lowercase letter, number, english, and special letter - _ . , : / @ # and length of key under 128."),
		},
		{
			// Failure Case - wrong expected value
			name:    "UnmarshalDeviceTwinUpdateTest-WrongExpectedValue",
			payload: bytesTwinKeyExpectedValueError,
			want:    &keyTwinUpdateExpectedValueError,
			wantErr: errors.New("The value of twin must be only include upper or lowercase letter, number, english, and special letter - _ . , : / @ # and length of key under 512."),
		},
		{
			// Failure Case - wrong actual value
			name:    "UnmarshalDeviceTwinUpdateTest-WrongActualValue",
			payload: bytesTwinKeyActualValueError,
			want:    &keyTwinUpdateActualValueError,
			wantErr: errors.New("The value of twin must be only include upper or lowercase letter, number, english, and special letter - _ . , : / @ # and length of key under 512."),
		},
		{
			// Success Case
			name:    "UnmarshalDeviceTwinUpdateTest-RightKeyFormat",
			payload: bytesTwinKey,
			want:    &keyTwinUpdate,
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnmarshalDeviceTwinUpdate(test.payload)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("UnmarshalDeviceTwinUpdate() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalDeviceTwinUpdate() = %v, want %v", got, test.want)
			}
		})
	}
}
