/*
Copyright 2024 The KubeEdge Authors.

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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	edgecon "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
)

func TestIsVolumeResource(t *testing.T) {
	assert := assert.New(t)

	validResource := "test/" + constants.CSIResourceTypeVolume + "/resource"
	invalidResource := "test/resourcePath/resource"

	assert.True(IsVolumeResource(validResource))
	assert.False(IsVolumeResource(invalidResource))
}

func TestGetMessageUID(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		name      string
		msg       beehivemodel.Message
		stdResult string
		hasError  bool
	}{
		{
			name: "Valid UID",
			msg: beehivemodel.Message{
				Content: &v1.ObjectMeta{
					UID: "test-uid",
				},
			},
			stdResult: "test-uid",
			hasError:  false,
		},
		{
			name: "Invalid content type",
			msg: beehivemodel.Message{
				Content: "",
			},
			stdResult: "",
			hasError:  true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := GetMessageUID(test.msg)
			if test.hasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			assert.Equal(test.stdResult, result)
		})
	}
}

func TestGetMessageDeletionTimestamp(t *testing.T) {
	assert := assert.New(t)

	now := v1.Now()
	cases := []struct {
		name      string
		msg       beehivemodel.Message
		stdResult *v1.Time
		hasError  bool
	}{
		{
			name: "Valid DeletionTimestamp",
			msg: beehivemodel.Message{
				Content: &v1.ObjectMeta{
					DeletionTimestamp: &now,
				},
			},
			stdResult: &now,
			hasError:  false,
		},
		{
			name: "Invalid content type",
			msg: beehivemodel.Message{
				Content: "",
			},
			stdResult: nil,
			hasError:  true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := GetMessageDeletionTimestamp(&test.msg)
			if test.hasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			assert.Equal(test.stdResult, result)
		})
	}
}

func TestTrimMessage(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		name      string
		resource  string
		stdResult string
	}{
		{
			name:      "Valid resource",
			resource:  "node/test-node/namespace/pod/test-pod",
			stdResult: "namespace/pod/test-pod",
		},
		{
			name:      "Invalid length of resource",
			resource:  "node/nodeName",
			stdResult: "node/nodeName",
		},
		{
			name:      "Resource is not starting with node",
			resource:  "namespace/pod/test-pod-two",
			stdResult: "namespace/pod/test-pod-two",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			msg := beehivemodel.NewMessage("")
			msg.SetResourceOperation(test.resource, "operation")
			TrimMessage(msg)
			assert.Equal(test.stdResult, msg.GetResource())
		})
	}
}

func TestConstructConnectMessage(t *testing.T) {
	assert := assert.New(t)

	nodeID := "test-node-id"
	info := &model.HubInfo{NodeID: nodeID}

	msg := ConstructConnectMessage(info, true)
	assert.NotNil(msg)
	assert.Equal(model.SrcCloudHub, msg.GetSource())
	assert.Equal(model.GpResource, msg.GetGroup())
	assert.Equal(model.NewResource(model.ResNode, nodeID, nil), msg.GetResource())

	var body map[string]interface{}
	err := json.Unmarshal(msg.GetContent().([]byte), &body)
	assert.NoError(err)
	assert.Equal(model.OpConnect, body["event_type"])
	assert.Equal(nodeID, body["client_id"])
}

func TestDeepCopy(t *testing.T) {
	assert := assert.New(t)

	msg := beehivemodel.NewMessage("sample message")
	msg.FillBody("sample content")

	dc := DeepCopy(msg)
	assert.NotNil(dc)
	assert.Equal(msg.GetID(), dc.GetID())
	assert.Equal(msg.GetContent(), dc.GetContent())
}

func TestAckMessageKeyFunc(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		name      string
		obj       interface{}
		stdResult string
		hasError  bool
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
			stdResult: "test-uid",
			hasError:  false,
		},
		{
			name:      "Invalid object type",
			obj:       "invalid",
			stdResult: "",
			hasError:  true,
		},
		{
			name: "Message without GroupResource",
			obj: &beehivemodel.Message{
				Header: beehivemodel.MessageHeader{ID: "test-id"},
				Router: beehivemodel.MessageRoute{Group: "other-group"},
			},
			stdResult: "",
			hasError:  true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := AckMessageKeyFunc(test.obj)
			if test.hasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			assert.Equal(test.stdResult, result)
		})
	}
}

func TestNoAckMessageKeyFunc(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		name      string
		obj       interface{}
		stdResult string
		hasError  bool
	}{
		{
			name: "Valid message",
			obj: &beehivemodel.Message{
				Header: beehivemodel.MessageHeader{ID: "test-id"},
			},
			stdResult: "test-id",
			hasError:  false,
		},
		{
			name:      "Invalid object type",
			obj:       "invalid",
			stdResult: "",
			hasError:  true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := NoAckMessageKeyFunc(test.obj)
			if test.hasError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			assert.Equal(test.stdResult, result)
		})
	}
}
