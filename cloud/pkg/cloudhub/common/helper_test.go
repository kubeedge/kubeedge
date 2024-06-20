/*
Copyright 2022 The KubeEdge Authors.

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

package common

import (
	"testing"
	"encoding/json"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	edgecon "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
)

func TestIsVolumeResource(t *testing.T){
	assert := assert.New(t)

	validVolumeResource := "test/" + constants.CSIResourceTypeVolume + "/resource"
	invalidVolumeResource := "test/resourcePath/resource"

	assert.True(IsVolumeResource(validVolumeResource))
	assert.False(IsVolumeResource(invalidVolumeResource))
}

func TestGetMessageUID(t *testing.T) {
	tests := []struct {
		name     string
		msg      beehivemodel.Message
		stdResult string
		hasError bool
	}{
		{
			name: "Valid UID",
			msg: beehivemodel.Message{
				Content: &v1.ObjectMeta{
					UID: "test-one",
				},
			},
			stdResult: "test-one",
			hasError: false,
		},
		{
			name: "Invalid content type",
			msg: beehivemodel.Message{
				Content: "",
			},
			stdResult: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetMessageUID(tt.msg)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.stdResult, result)
		})
	}
}

func TestGetMessageDeletionTimestamp(t *testing.T) {
	now := v1.Now()
	tests := []struct {
		name     string
		msg      beehivemodel.Message
		stdResult *v1.Time
		hasError bool
	}{
		{
			name: "Valid DeletionTimestamp",
			msg: beehivemodel.Message{
				Content: &v1.ObjectMeta{
					DeletionTimestamp: &now,
				},
			},
			stdResult: &now,
			hasError: false,
		},
		{
			name: "Invalid content type",
			msg: beehivemodel.Message{
				Content: "",
			},
			stdResult: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetMessageDeletionTimestamp(&tt.msg)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.stdResult, result)
		})
	}
}

func TestTrimMessage(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		stdResult string
	}{
		{
			name:     "Valid resource",
			resource: "node/test-node/namespace/pod/test-pod",
			stdResult: "namespace/pod/test-pod",
		},
		{
			name:     "Invalid resource length",
			resource: "node/nodeName",
			stdResult: "node/nodeName",
		},
		{
			name:     "Resource not starting with node",
			resource: "namespace/pod/test-pod-two",
			stdResult: "namespace/pod/test-pod-two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := beehivemodel.NewMessage("")
			msg.SetResourceOperation(tt.resource, "operation")
			TrimMessage(msg)
			assert.Equal(t, tt.stdResult, msg.GetResource())
		})
	}
}

func TestConstructConnectMessage(t *testing.T) {
	nodeID := "node123"
	info := &model.HubInfo{NodeID: nodeID}

	msg := ConstructConnectMessage(info, true)
	assert.NotNil(t, msg)
	assert.Equal(t, model.SrcCloudHub, msg.GetSource())
	assert.Equal(t, model.GpResource, msg.GetGroup())
	assert.Equal(t, model.NewResource(model.ResNode, nodeID, nil), msg.GetResource())

	var body map[string]interface{}
	err := json.Unmarshal(msg.GetContent().([]byte), &body)
	assert.NoError(t, err)
	assert.Equal(t, model.OpConnect, body["event_type"])
	assert.Equal(t, nodeID, body["client_id"])
}

func TestDeepCopy(t *testing.T) {
	original := beehivemodel.NewMessage("test")
	original.FillBody("content")

	copy := DeepCopy(original)
	assert.NotNil(t, copy)
	assert.Equal(t, original.GetID(), copy.GetID())
	assert.Equal(t, original.GetContent(), copy.GetContent())
}

func TestAckMessageKeyFunc(t *testing.T) {
	tests := []struct {
		name     string
		obj      interface{}
		expected string
		hasError bool
	}{
		{
			name: "Valid message with GroupResource",
			obj: &beehivemodel.Message{
				Header: beehivemodel.MessageHeader{ID: "test-id"},
				Router: beehivemodel.MessageRoute{Group: edgecon.GroupResource},
				Content: &v1.ObjectMeta{
					UID: "test-uid",
				},
			},
			expected: "test-uid",
			hasError: false,
		},
		{
			name:     "Invalid object type",
			obj:      "invalid",
			expected: "",
			hasError: true,
		},
		{
			name: "Message without GroupResource",
			obj: &beehivemodel.Message{
				Header: beehivemodel.MessageHeader{ID: "test-id"},
				Router: beehivemodel.MessageRoute{Group: "other-group"},
			},
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AckMessageKeyFunc(tt.obj)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNoAckMessageKeyFunc(t *testing.T) {
	tests := []struct {
		name     string
		obj      interface{}
		expected string
		hasError bool
	}{
		{
			name: "Valid message",
			obj: &beehivemodel.Message{
				Header: beehivemodel.MessageHeader{ID: "test-id"},
			},
			expected: "test-id",
			hasError: false,
		},
		{
			name:     "Invalid object type",
			obj:      "invalid",
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NoAckMessageKeyFunc(tt.obj)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

