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

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
)

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

// createDeviceTwin() is function to create an array of DeviceTwin.
func createDeviceTwin(devTwin dtclient.DeviceTwin) []dtclient.DeviceTwin {
	deviceTwin := []dtclient.DeviceTwin{}
	deviceTwin = append(deviceTwin, devTwin)
	return deviceTwin
}

// TestBuildMembershipGetResult is function to test BuildMembershipGetResult().
func TestBuildMembershipGetResult(t *testing.T) {
	baseMessage := BaseMessage{EventID: uuid.New().String(), Timestamp: time.Now().UnixNano() / 1e6}
	devices := []*v1alpha2.Device{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "id1",
				Namespace: "default",
			},
		},
	}

	bytesMemGetResult, _ := json.Marshal(devices)
	tests := []struct {
		name        string
		baseMessage BaseMessage
		devices     []*v1alpha2.Device
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
			got, err := BuildMembershipGetResult(test.devices)
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
func createMessageTwin() []v1alpha2.Twin {
	msgTwins := make([]v1alpha2.Twin, 0)
	twinMetadataDeleted := v1alpha2.Twin{
		PropertyName: "deleted",
		Desired: v1alpha2.TwinProperty{
			Metadata: map[string]string{
				"type": "deleted",
			},
		},
	}

	twinMetadataUpdated := v1alpha2.Twin{
		PropertyName: "updated",
		Desired: v1alpha2.TwinProperty{
			Metadata: map[string]string{
				"type": "updated",
			},
		},
	}

	nilTwin := v1alpha2.Twin{
		PropertyName: "empty",
	}
	msgTwins = append(msgTwins, twinMetadataDeleted, twinMetadataUpdated, nilTwin)
	return msgTwins
}

// createDeviceTwinResultDealTypeGet() is function to create DeviceTwinResult with DealType 0(Get).
func createDeviceTwinResultDealTypeGet(deviceName string) v1alpha2.Device {
	device := v1alpha2.Device{}
	device.Name = deviceName
	device.Namespace = "default"
	device.Status.Twins = make([]v1alpha2.Twin, 0)

	twinMetadataUpdated := v1alpha2.Twin{
		PropertyName: "updated",
		Desired: v1alpha2.TwinProperty{
			Metadata: map[string]string{
				"type": "updated",
			},
		},
	}

	twinMetadataEmpty := v1alpha2.Twin{
		PropertyName: "empty",
	}
	device.Status.Twins = append(device.Status.Twins, twinMetadataUpdated, twinMetadataEmpty)

	return device
}

// createDeviceTwinResult() is function to create DeviceTwinResult with DealType of value other than 0(1-Update, 2-Sync).
func createDeviceTwinResult(deviceName string) v1alpha2.Device {
	device := v1alpha2.Device{}
	device.Namespace = "default"
	device.Name = deviceName

	msgTwins := createMessageTwin()
	device.Status.Twins = msgTwins
	return device
}

// TestBuildDeviceTwinResult is function to test BuildDeviceTwinResult().
func TestBuildDeviceTwinResult(t *testing.T) {
	msgTwins := []v1alpha2.Twin{
		{
			PropertyName: "deleted",
			Desired: v1alpha2.TwinProperty{
				Metadata: map[string]string{
					"type": "deleted",
				},
			},
		},
		{
			PropertyName: "updated",
			Desired: v1alpha2.TwinProperty{
				Metadata: map[string]string{
					"type": "updated",
				},
			},
		},
		{
			PropertyName: "empty",
		},
	}

	devTwinResultDealType0 := createDeviceTwinResultDealTypeGet("Test1")
	bytesDealType0, _ := json.Marshal(devTwinResultDealType0)
	devTwinResult1 := createDeviceTwinResult("Test2")
	bytesDealType1, _ := json.Marshal(devTwinResult1)
	tests := []struct {
		name     string
		twins    []v1alpha2.Twin
		dealType int
		want     []byte
		wantErr  error
	}{
		{
			name:     "Test1",
			twins:    msgTwins,
			dealType: 0,
			want:     bytesDealType0,
			wantErr:  nil,
		},
		{
			name:     "Test2",
			twins:    msgTwins,
			dealType: 1,
			want:     bytesDealType1,
			wantErr:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := BuildDeviceTwinResult("default"+"/"+test.name, test.twins, test.dealType)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("BuildDeviceTwinResult() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("BuildDeviceTwinResult() = %v, want %v", string(got), string(test.want))
			}
		})
	}
}

// createMessageTwinWithDiffValues() is function to create MessageTwin with actual and expected values.
func createMessageTwinWithDiffValues(baseMessage BaseMessage) []v1alpha2.Twin {
	msgTwins := make([]v1alpha2.Twin, 0)
	twinMetadataDeleted := v1alpha2.Twin{
		PropertyName: "deleted",
		Desired: v1alpha2.TwinProperty{
			Metadata: map[string]string{
				"type": "deleted",
			},
		},
	}

	msgTwins = append(msgTwins, twinMetadataDeleted)

	expected := "ON"
	actual := "OFF"
	twinActualExpected := v1alpha2.Twin{
		PropertyName: "twin",
		Desired: v1alpha2.TwinProperty{
			Value: expected,
			Metadata: map[string]string{
				"type": "updated",
			},
		},
		Reported: v1alpha2.TwinProperty{
			Value: actual,
		},
	}
	msgTwins = append(msgTwins, twinActualExpected)

	twinExpected := v1alpha2.Twin{
		PropertyName: "expected",
		Desired: v1alpha2.TwinProperty{
			Value: expected,
			Metadata: map[string]string{
				"type": "updated",
			},
		},
	}
	msgTwins = append(msgTwins, twinExpected)

	twinActual := v1alpha2.Twin{
		PropertyName: "actual",
		Reported: v1alpha2.TwinProperty{
			Value: expected,
			Metadata: map[string]string{
				"type": "updated",
			},
		},
	}
	msgTwins = append(msgTwins, twinActual)

	return msgTwins
}

// createMessageTwinWithSameValues() is function to create MessageTwin with same actual and expected values.
func createMessageTwinWithSameValues() []v1alpha2.Twin {
	value := "ON"
	msgTwin := make([]v1alpha2.Twin, 0)
	twins := v1alpha2.Twin{
		PropertyName: "twins",
		Reported: v1alpha2.TwinProperty{
			Value: value,
		},
		Desired: v1alpha2.TwinProperty{
			Value: value,
			Metadata: map[string]string{
				"type": "updated",
			},
		},
	}

	msgTwin = append(msgTwin, twins)
	return msgTwin
}

// createMessageTwinAndDeltaWithDiffValues is function to create MessageTwin and Delta with different actual and expected values.
func createMessageTwinAndDeltaWithDiffValues() v1alpha2.Device {
	delta := v1alpha2.Device{}
	expected := "ON"
	actual := "OFF"
	twinActualExpected := v1alpha2.Twin{
		PropertyName: "twin",
		Desired: v1alpha2.TwinProperty{
			Value: expected,
			Metadata: map[string]string{
				"type": "updated",
			},
		},
		Reported: v1alpha2.TwinProperty{
			Value: actual,
		},
	}

	twinExpected := v1alpha2.Twin{
		PropertyName: "expected",
		Desired: v1alpha2.TwinProperty{
			Value: expected,
			Metadata: map[string]string{
				"type": "updated",
			},
		},
	}

	delta.Status.Twins = make([]v1alpha2.Twin, 0)
	delta.Status.Twins = append(delta.Status.Twins, twinActualExpected, twinExpected)

	return delta
}

// createMessageTwinAndDeltaWithSameValues is function to create MessageTwin and Delta with same actual and expected values.
func createMessageTwinAndDeltaWithSameValues() v1alpha2.Device {
	deltas := v1alpha2.Device{}
	resultTwins := make([]v1alpha2.Twin, 0)
	deltas.Status.Twins = resultTwins
	return deltas
}

// TestBuildDeviceTwinDelta is function to test BuildDeviceTwinDelta().
func TestBuildDeviceTwinDelta(t *testing.T) {
	baseMessage := BaseMessage{EventID: "Event1", Timestamp: time.Now().UnixNano() / 1e6}
	msgTwins := createMessageTwinWithDiffValues(baseMessage)

	delta := createMessageTwinAndDeltaWithDiffValues()
	bytesResultTwinDiffValues, _ := json.Marshal(delta)
	msgTwin := createMessageTwinWithSameValues()
	deltas := createMessageTwinAndDeltaWithSameValues()
	bytesResultTwinSameValues, _ := json.Marshal(deltas)
	tests := []struct {
		name        string
		baseMessage BaseMessage
		twins       []v1alpha2.Twin
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
			got, got1 := BuildDeviceTwinDelta(test.twins)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("BuildDeviceTwinDelta() got = %v, want %v", string(got), string(test.want))
			}
			if got1 != test.wantBool {
				t.Errorf("BuildDeviceTwinDelta() got1 = %v, want %v", got1, test.wantBool)
			}
		})
	}
}
