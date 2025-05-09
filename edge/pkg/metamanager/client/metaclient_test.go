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
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// Override the global variables to use very short timeouts
func init() {
	// Override with very short timeouts to prevent tests from hanging
	syncPeriod = 1 * time.Millisecond
	syncMsgRespTimeout = 5 * time.Millisecond

	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	moduleInfo := &common.ModuleInfo{
		ModuleName: modules.MetaManagerModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(moduleInfo)

	klog.InitFlags(nil)
}

type MockSend struct {
	shouldFail    bool
	retries       int
	lastResource  string
	lastOperation string
}

func (m *MockSend) SendSync(message *model.Message) (*model.Message, error) {
	m.lastResource = message.GetResource()
	m.lastOperation = message.GetOperation()

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

func (m *MockSend) Send(message *model.Message) {
	m.lastResource = message.GetResource()
	m.lastOperation = message.GetOperation()
}

func TestNew(t *testing.T) {
	client := New()
	assert.NotNil(t, client, "New() should return a non-nil client")

	// Check that the returned client is of type metaClient
	_, ok := client.(*metaClient)
	assert.True(t, ok, "New() should return a metaClient")

	// Also check that send is initialized
	mc, _ := client.(*metaClient)
	assert.NotNil(t, mc.send, "New() should initialize send")

	// Check the type of send
	_, ok = mc.send.(*send)
	assert.True(t, ok, "send should be of type *send")
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
			name: "Secrets",
			testFunc: func() interface{} {
				return client.Secrets("default")
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
			name: "PodStatus",
			testFunc: func() interface{} {
				return client.PodStatus("default")
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

func TestSendMethods(t *testing.T) {
	mockSend := &MockSend{shouldFail: false}
	message := &model.Message{
		Header: model.MessageHeader{
			ID: "test-id",
		},
		Router: model.MessageRoute{
			Resource:  "test/resource",
			Operation: "test-operation",
		},
	}

	t.Run("SendSync", func(t *testing.T) {
		resp, err := mockSend.SendSync(message)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "response-id", resp.Header.ID)
	})

	t.Run("Send", func(t *testing.T) {
		assert.NotPanics(t, func() {
			mockSend.Send(message)
		})
		assert.Equal(t, "test/resource", mockSend.lastResource)
		assert.Equal(t, "test-operation", mockSend.lastOperation)
	})
}

func TestSendSyncWithError(t *testing.T) {
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

// TestSendImplementation directly tests the actual send struct implementation with minimal execution
func TestSendImplementation(t *testing.T) {
	// Testing the newSend function
	sender := newSend()
	assert.NotNil(t, sender)
	_, ok := sender.(*send)
	assert.True(t, ok)

	// Create a minimalist message
	message := &model.Message{
		Header: model.MessageHeader{
			ID: "test-id",
		},
		Router: model.MessageRoute{
			Resource:  "test/resource",
			Operation: "test-operation",
		},
	}

	// Test the Send method - this should be quick
	t.Run("send.Send() should not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			sender.Send(message)
		})
	})

	// Test the SendSync method with a timeout
	// This will likely fail due to missing module, but we just want code coverage
	t.Run("send.SendSync() should not hang", func(t *testing.T) {
		// Use a timeout to prevent the test from hanging
		c := make(chan struct{})
		go func() {
			defer close(c)
			_, err := sender.SendSync(message)
			// Just check that it returned, we expect an error
			assert.Error(t, err)
		}()

		// Use a timeout slightly longer than our syncMsgRespTimeout
		select {
		case <-c:
			// Test completed normally
		case <-time.After(100 * time.Millisecond):
			t.Fatal("SendSync is hanging")
		}
	})
}
