/*
Copyright 2025 The KubeEdge Authors.

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

package dmiserver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
)

func TestCreateMessageTwinUpdate(t *testing.T) {
	assert := assert.New(t)

	twin := &pb.Twin{
		PropertyName: "temperature",
		ObservedDesired: &pb.TwinProperty{
			Value: "25",
		},
		Reported: &pb.TwinProperty{
			Value: "24",
		},
	}

	msg, err := CreateMessageTwinUpdate(twin)
	assert.NoError(err)
	assert.NotEmpty(msg)

	var result DeviceTwinUpdate
	assert.NoError(json.Unmarshal(msg, &result))

	// Verify timestamp is set
	assert.Greater(result.BaseMessage.Timestamp, int64(0))

	// Safely check the twin entry exists before dereferencing
	tempTwin, ok := result.Twin["temperature"]
	if !ok || tempTwin == nil {
		t.Fatalf("expected twin entry for 'temperature' to exist and be non-nil")
	}

	if tempTwin.Expected == nil || tempTwin.Expected.Value == nil {
		t.Fatalf("expected Expected.Value to be non-nil")
	}
	assert.Equal("25", *tempTwin.Expected.Value)

	if tempTwin.Actual == nil || tempTwin.Actual.Value == nil {
		t.Fatalf("expected Actual.Value to be non-nil")
	}
	assert.Equal("24", *tempTwin.Actual.Value)

	// Verify exact JSON structure has the "twin" key
	var raw map[string]interface{}
	assert.NoError(json.Unmarshal(msg, &raw))
	_, hasTwin := raw["twin"]
	assert.True(hasTwin, "JSON must contain 'twin' key")
}

func TestCreateMessageTwinUpdateNilInput(t *testing.T) {
	// Production code is hardened to return an error rather than panic on nil input.
	_, err := CreateMessageTwinUpdate(nil)
	assert.Error(t, err, "expected error when twin is nil")
	assert.Contains(t, err.Error(), "nil")
}

func TestCreateMessageStateUpdate(t *testing.T) {
	assert := assert.New(t)

	req := &pb.ReportDeviceStatesRequest{
		DeviceName:      "device1",
		DeviceNamespace: "default",
		State:           "online",
	}

	msg, err := CreateMessageStateUpdate(req)
	assert.NoError(err)
	assert.NotEmpty(msg)

	var result DeviceStateUpdate
	assert.NoError(json.Unmarshal(msg, &result))

	// Verify timestamp is set
	assert.Greater(result.BaseMessage.Timestamp, int64(0))

	// Verify state value
	assert.Equal("online", result.State)

	// Verify the wire format uses lowercase "state" key (not "State")
	var raw map[string]interface{}
	assert.NoError(json.Unmarshal(msg, &raw))

	_, hasLower := raw["state"]
	_, hasUpper := raw["State"]
	assert.True(hasLower, "JSON must contain lowercase 'state' key")
	assert.False(hasUpper, "JSON must NOT contain uppercase 'State' key")
}

func TestCreateMessageStateUpdateNilInput(t *testing.T) {
	// Production code is hardened to return an error rather than panic on nil input.
	_, err := CreateMessageStateUpdate(nil)
	assert.Error(t, err, "expected error when request is nil")
	assert.Contains(t, err.Error(), "nil")
}
