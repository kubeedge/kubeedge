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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

const testRuntimeClassName = "kata-containers"

func createTestRuntimeClass() *nodev1.RuntimeClass {
	return &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: testRuntimeClassName,
		},
		Handler: "kata",
	}
}

func TestNewRuntimeClasses(t *testing.T) {
	assert := assert.New(t)

	s := newSend()
	rc := newRuntimeClasses(s)

	assert.NotNil(rc)
	assert.IsType(&send{}, rc.send)
}

// TestRuntimeClass_Get_FromMetaDB exercises the local-cache (MetaDB) fast path.
// The metaService.QueryMeta returns pre-populated data, so SendSync is never called.
func TestRuntimeClass_Get_FromMetaDB(t *testing.T) {
	assert := assert.New(t)

	expectedRC := createTestRuntimeClass()
	rcJSON, _ := json.Marshal(expectedRC)

	mockMeta := &MockMetaService{
		QueryMetaFunc: func(key, value string) (*[]string, error) {
			return &[]string{string(rcJSON)}, nil
		},
	}

	// SendSync must NOT be called when MetaDB has data.
	mockSend := &mockSendInterface{
		sendSyncFunc: func(_ *model.Message) (*model.Message, error) {
			t.Error("SendSync should not be called when MetaDB has data")
			return nil, fmt.Errorf("unexpected call")
		},
	}

	rcClient := NewRuntimeClassesWithMetaService(mockSend, mockMeta)
	rc, err := rcClient.Get(testRuntimeClassName)

	assert.NoError(err)
	assert.Equal(expectedRC, rc)
}

// TestRuntimeClass_Get_RemoteGet exercises the remote (MetaManager) fallback path.
func TestRuntimeClass_Get_RemoteGet(t *testing.T) {
	assert := assert.New(t)

	expectedRC := createTestRuntimeClass()
	rcJSON, _ := json.Marshal(expectedRC)

	resource := fmt.Sprintf("%s/%s/%s", models.NullNamespace, model.ResourceTypeRuntimeClass, testRuntimeClassName)

	testCases := []struct {
		name        string
		queryErr    error
		queryResult *[]string
		respFunc    func(*model.Message) (*model.Message, error)
		expectedRC  *nodev1.RuntimeClass
		expectErr   bool
		errContains string
	}{
		{
			name:        "Get RuntimeClass from MetaManager (DB miss)",
			queryResult: &[]string{}, // empty → triggers remoteGet
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Source = "other-module"
				resp.Content = rcJSON
				return resp, nil
			},
			expectedRC: expectedRC,
			expectErr:  false,
		},
		{
			name:        "Get RuntimeClass from MetaManager (DB error)",
			queryErr:    fmt.Errorf("db error"),
			queryResult: nil,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = rcJSON
				return resp, nil
			},
			expectedRC: expectedRC,
			expectErr:  false,
		},
		{
			name:        "SendSync Error",
			queryResult: &[]string{},
			respFunc: func(_ *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("send sync error")
			},
			expectedRC:  nil,
			expectErr:   true,
			errContains: "get runtimeclass from metaManager failed",
		},
		{
			name:        "Content error (msg.GetContent is error)",
			queryResult: &[]string{},
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = fmt.Errorf("some upstream error")
				return resp, nil
			},
			expectedRC:  nil,
			expectErr:   true,
			errContains: "some upstream error",
		},
		{
			name:        "Content Unmarshal Error from MetaManager",
			queryResult: &[]string{},
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = []byte(`{"invalid": json}`)
				return resp, nil
			},
			expectedRC:  nil,
			expectErr:   true,
			errContains: "unmarshal message to runtimeclass failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal(resource, message.GetResource())
				assert.Equal(model.QueryOperation, message.GetOperation())
				return tc.respFunc(message)
			}

			mockMeta := &MockMetaService{
				QueryMetaFunc: func(key, value string) (*[]string, error) {
					return tc.queryResult, tc.queryErr
				},
			}

			rcClient := NewRuntimeClassesWithMetaService(mockSend, mockMeta)
			rc, err := rcClient.Get(testRuntimeClassName)

			if tc.expectErr {
				assert.Error(err)
				if tc.errContains != "" {
					assert.Contains(err.Error(), tc.errContains)
				}
				assert.Nil(rc)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedRC, rc)
			}
		})
	}
}

func TestHandleRuntimeClassFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	rc := createTestRuntimeClass()
	rcJSON, _ := json.Marshal(rc)

	// Valid single-item list.
	validList := []string{string(rcJSON)}
	result, err := handleRuntimeClassFromMetaDB(&validList)
	assert.NoError(err)
	assert.Equal(rc, result)

	// Empty list → error.
	emptyList := []string{}
	result, err = handleRuntimeClassFromMetaDB(&emptyList)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "runtimeclass length from meta db is 0")

	// Multiple items → error.
	multiList := []string{string(rcJSON), string(rcJSON)}
	result, err = handleRuntimeClassFromMetaDB(&multiList)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "runtimeclass length from meta db is 2")

	// Invalid JSON → error.
	invalidList := []string{`{"invalid": json}`}
	result, err = handleRuntimeClassFromMetaDB(&invalidList)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to runtimeclass from db failed")
}

func TestHandleRuntimeClassFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	rc := createTestRuntimeClass()
	content, _ := json.Marshal(rc)

	// Valid JSON.
	result, err := handleRuntimeClassFromMetaManager(content)
	assert.NoError(err)
	assert.Equal(rc, result)

	// Empty object.
	result, err = handleRuntimeClassFromMetaManager([]byte("{}"))
	assert.NoError(err)
	assert.Equal(&nodev1.RuntimeClass{}, result)

	// Invalid JSON.
	result, err = handleRuntimeClassFromMetaManager([]byte(`{"invalid": json}`))
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to runtimeclass failed")
}
