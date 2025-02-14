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

package client

import (
	"fmt"
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const testNodeName = "test-node"

func TestNewNodeStatus(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	ns := newNodeStatus(namespace, s)

	assert.NotNil(ns)
	assert.Equal(namespace, ns.namespace)
	assert.IsType(&mockSend{}, ns.send)
}

func TestNodeStatus_Create(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	ns := newNodeStatus(namespace, s)

	req := &edgeapi.NodeStatusRequest{
		UID: types.UID(testNodeName),
		Status: v1.NodeStatus{
			Phase: v1.NodeRunning,
		},
		ExtendResources: map[v1.ResourceName][]edgeapi.ExtendResource{},
	}

	resp, err := ns.Create(req)
	assert.Nil(resp)
	assert.NoError(err)
}

func TestNodeStatus_Update(t *testing.T) {
	testCases := []struct {
		name        string
		rsName      string
		request     edgeapi.NodeStatusRequest
		mockSetup   func(*mockSend)
		expectError bool
	}{
		{
			name:   "successful update",
			rsName: testNodeName,
			request: edgeapi.NodeStatusRequest{
				UID: types.UID(testNodeName),
				Status: v1.NodeStatus{
					Phase: v1.NodeRunning,
				},
				ExtendResources: map[v1.ResourceName][]edgeapi.ExtendResource{},
			},
			mockSetup: func(m *mockSend) {
				m.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					return &model.Message{}, nil
				}
			},
			expectError: false,
		},
		{
			name:   "update error",
			rsName: testNodeName,
			request: edgeapi.NodeStatusRequest{
				UID:             types.UID(testNodeName),
				Status:          v1.NodeStatus{},
				ExtendResources: map[v1.ResourceName][]edgeapi.ExtendResource{},
			},
			mockSetup: func(m *mockSend) {
				m.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					return nil, fmt.Errorf("update failed")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mock := newMockSend()
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			ns := newNodeStatus(namespace, mock)
			err := ns.Update(tt.rsName, tt.request)

			if tt.expectError {
				assert.Error(err)
				assert.Contains(err.Error(), "update nodeStatus failed")
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestNodeStatus_Delete(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	ns := newNodeStatus(namespace, s)

	err := ns.Delete(testNodeName)
	// Currently returns nil as per implementation
	assert.NoError(err)
}

func TestNodeStatus_Get(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	ns := newNodeStatus(namespace, s)

	resp, err := ns.Get(testNodeName)
	// Currently returns nil, nil as per implementation
	assert.Nil(resp)
	assert.NoError(err)
}
