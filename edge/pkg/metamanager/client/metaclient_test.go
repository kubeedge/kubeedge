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

package client

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

type MockSend struct {
	shouldFail bool
	retries    int
}

func (m *MockSend) SendSync(_ *model.Message) (*model.Message, error) {
	if m.shouldFail {
		m.retries++
		return nil, errors.New("mock error")
	}
	resp := &model.Message{
		Header: model.MessageHeader{
			ID: "response-id",
		},
	}
	return resp, nil
}

func (m *MockSend) Send(_ *model.Message) {
}

func TestNew(t *testing.T) {
	client := New()
	assert.NotNil(t, client, "New() should return a non-nil client")
}

func TestMetaClientInterfaces(t *testing.T) {
	mockSend := &MockSend{}
	client := &metaClient{send: mockSend}

	tests := []struct {
		name     string
		testFunc func() interface{}
	}{
		{
			name: "Pods",
			testFunc: func() interface{} {
				return client.Pods("default")
			},
		},
		{
			name: "ConfigMaps",
			testFunc: func() interface{} {
				return client.ConfigMaps("default")
			},
		},
		{
			name: "Events",
			testFunc: func() interface{} {
				return client.Events("default")
			},
		},
		{
			name: "Nodes",
			testFunc: func() interface{} {
				return client.Nodes("default")
			},
		},
		{
			name: "NodeStatus",
			testFunc: func() interface{} {
				return client.NodeStatus("default")
			},
		},
		{
			name: "Secrets",
			testFunc: func() interface{} {
				return client.Secrets("default")
			},
		},
		{
			name: "ServiceAccountToken",
			testFunc: func() interface{} {
				return client.ServiceAccountToken()
			},
		},
		{
			name: "ServiceAccounts",
			testFunc: func() interface{} {
				return client.ServiceAccounts("default")
			},
		},
		{
			name: "PodStatus",
			testFunc: func() interface{} {
				return client.PodStatus("default")
			},
		},
		{
			name: "PersistentVolumes",
			testFunc: func() interface{} {
				return client.PersistentVolumes()
			},
		},
		{
			name: "PersistentVolumeClaims",
			testFunc: func() interface{} {
				return client.PersistentVolumeClaims("default")
			},
		},
		{
			name: "VolumeAttachments",
			testFunc: func() interface{} {
				return client.VolumeAttachments("default")
			},
		},
		{
			name: "Leases",
			testFunc: func() interface{} {
				return client.Leases("default")
			},
		},
		{
			name: "CertificateSigningRequests",
			testFunc: func() interface{} {
				return client.CertificateSigningRequests()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.testFunc()
			assert.NotNil(t, result, "Interface %s should not return nil", tt.name)
		})
	}
}

func TestSendSync(t *testing.T) {
	testCases := []struct {
		name          string
		shouldFail    bool
		expectedError bool
	}{
		{
			name:          "successful send",
			shouldFail:    false,
			expectedError: false,
		},
		{
			name:          "failed send with retries",
			shouldFail:    true,
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSend := &MockSend{shouldFail: tc.shouldFail}
			message := &model.Message{
				Header: model.MessageHeader{
					ID: "test-id",
				},
			}

			resp, err := mockSend.SendSync(message)

			if tc.expectedError {
				assert.Error(t, err)
				assert.True(t, mockSend.retries > 0, "Should have attempted retries")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, "response-id", resp.Header.ID)
			}
		})
	}
}

func TestSend(t *testing.T) {
	sender := newSend()
	message := &model.Message{
		Header: model.MessageHeader{
			ID: "test-id",
		},
	}

	assert.NotPanics(t, func() {
		sender.Send(message)
	})
}

func init() {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	moduleInfo := &common.ModuleInfo{
		ModuleName: modules.MetaManagerModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(moduleInfo)

	klog.InitFlags(nil)
}
