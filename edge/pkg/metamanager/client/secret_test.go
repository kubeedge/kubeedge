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
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

const (
	testSecretName      = "test-secret"
	testSecretNamespace = "test-namespace"
)

func TestNewSecrets(t *testing.T) {
	assert := assert.New(t)

	s := newSend()

	secret := newSecrets(testSecretNamespace, s)

	assert.NotNil(secret)
	assert.Equal(testSecretNamespace, secret.namespace)
	assert.IsType(&send{}, secret.send)
}

func TestSecret_Create(t *testing.T) {
	assert := assert.New(t)

	inputSecret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testSecretNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("password123"),
		},
		Type: api.SecretTypeOpaque,
	}

	secretClient := newSecrets(testSecretNamespace, nil)
	createdSecret, err := secretClient.Create(inputSecret)

	assert.Nil(err, "Create method should return nil error")
	assert.Nil(createdSecret, "Create method should return nil secret")
}

func TestSecret_Update(t *testing.T) {
	assert := assert.New(t)

	inputSecret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testSecretNamespace,
		},
	}

	secretClient := newSecrets(testSecretNamespace, nil)
	err := secretClient.Update(inputSecret)

	assert.Nil(err, "Update method should return nil error")
}

func TestSecret_Delete(t *testing.T) {
	assert := assert.New(t)

	secretClient := newSecrets(testSecretNamespace, nil)
	err := secretClient.Delete(testSecretName)

	assert.Nil(err, "Delete method should return nil error")
}

func TestSecret_Get(t *testing.T) {
	assert := assert.New(t)

	expectedSecret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testSecretNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("password123"),
		},
		Type: api.SecretTypeOpaque,
	}
	secretJSON, err := json.Marshal(expectedSecret)
	assert.NoError(err, "Failed to marshal expected secret")
	metaDBList := []string{string(secretJSON)}

	resource := fmt.Sprintf("%s/%s/%s", testSecretNamespace, model.ResourceTypeSecret, testSecretName)

	testCases := []struct {
		name           string
		mockDBResult   *[]string
		mockDBError    error
		mockSendResult func(*model.Message) (*model.Message, error)
		expectedSecret *api.Secret
		expectErr      bool
		errContains    string
	}{
		{
			name:           "Get Secret from MetaDB success",
			mockDBResult:   &metaDBList,
			mockDBError:    nil,
			mockSendResult: nil,
			expectedSecret: expectedSecret,
			expectErr:      false,
		},
		{
			name:         "MetaDB error, remote Get success",
			mockDBResult: nil,
			mockDBError:  errors.New("database error"),
			mockSendResult: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = secretJSON
				return resp, nil
			},
			expectedSecret: expectedSecret,
			expectErr:      false,
		},
		{
			name:         "MetaDB empty, remote Get success",
			mockDBResult: &[]string{},
			mockDBError:  nil,
			mockSendResult: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = secretJSON
				return resp, nil
			},
			expectedSecret: expectedSecret,
			expectErr:      false,
		},
		{
			name:         "MetaDB error, remote SendSync error",
			mockDBResult: nil,
			mockDBError:  errors.New("database error"),
			mockSendResult: func(message *model.Message) (*model.Message, error) {
				return nil, errors.New("send sync error")
			},
			expectedSecret: nil,
			expectErr:      true,
			errContains:    "get secret from metaManager failed",
		},
		{
			name:         "MetaDB error, remote returns error content",
			mockDBResult: nil,
			mockDBError:  errors.New("database error"),
			mockSendResult: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = errors.New("error from remote")
				return resp, nil
			},
			expectedSecret: nil,
			expectErr:      true,
			errContains:    "error from remote",
		},
		{
			name:         "MetaDB error, remote content unmarshal error",
			mockDBResult: nil,
			mockDBError:  errors.New("database error"),
			mockSendResult: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = 123
				return resp, nil
			},
			expectedSecret: nil,
			expectErr:      true,
			errContains:    "unmarshal message to secret failed",
		},
		{
			name:         "MetaDB error, invalid JSON from remote",
			mockDBResult: nil,
			mockDBError:  errors.New("database error"),
			mockSendResult: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = []byte(`{"invalid": json}`)
				return resp, nil
			},
			expectedSecret: nil,
			expectErr:      true,
			errContains:    "unmarshal message to secret failed",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			patchQueryMeta := gomonkey.ApplyFunc(dao.QueryMeta, func(key string, value string) (*[]string, error) {
				assert.Equal("key", key)
				assert.Equal(resource, value)
				return test.mockDBResult, test.mockDBError
			})
			defer patchQueryMeta.Reset()

			mockSend := &mockSendInterface{}
			if test.mockSendResult != nil {
				mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
					assert.Equal(modules.MetaGroup, message.GetGroup())
					assert.Equal(modules.EdgedModuleName, message.GetSource())
					assert.NotEmpty(message.GetID())
					assert.Equal(resource, message.GetResource())
					assert.Equal(model.QueryOperation, message.GetOperation())

					return test.mockSendResult(message)
				}
			} else {
				mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
					t.Error("SendSync should not be called when getting from MetaDB")
					return nil, nil
				}
			}

			secretClient := newSecrets(testSecretNamespace, mockSend)
			secret, err := secretClient.Get(testSecretName)

			if test.expectErr {
				assert.Error(err)
				if test.errContains != "" {
					assert.Contains(err.Error(), test.errContains)
				}
				assert.Nil(secret)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedSecret, secret)
			}
		})
	}
}

func TestHandleSecretFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid Secret JSON in a single-element list
	secret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: "default",
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("password123"),
		},
		Type: api.SecretTypeOpaque,
	}
	secretJSON, _ := json.Marshal(secret)
	content := []string{string(secretJSON)}

	result, err := handleSecretFromMetaDB(&content)
	assert.NoError(err)
	assert.Equal(secret, result)

	// Test case 2: Empty list
	var emptyList []string

	result, err = handleSecretFromMetaDB(&emptyList)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "secret length from meta db is 0")

	// Test case 3: List with multiple elements
	multipleSecrets := []string{string(secretJSON), string(secretJSON)}

	result, err = handleSecretFromMetaDB(&multipleSecrets)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "secret length from meta db is 2")

	// Test case 4: Invalid JSON in the list
	invalidJSON := []string{"{invalid json}"}

	result, err = handleSecretFromMetaDB(&invalidJSON)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to secret from db failed")
}

func TestHandleSecretFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid Secret JSON
	secret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: "default",
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("password123"),
		},
		Type: api.SecretTypeOpaque,
	}
	content, _ := json.Marshal(secret)

	result, err := handleSecretFromMetaManager(content)
	assert.NoError(err)
	assert.Equal(secret, result)

	// Test case 2: Empty JSON
	emptyContent := []byte("{}")

	result, err = handleSecretFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.Equal(&api.Secret{}, result)

	// Test case 3: Invalid JSON
	invalidContent := []byte(`{"invalid": json}`)

	result, err = handleSecretFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to secret failed")

	// Test case 4: Partial Secret JSON
	partialSecret := []byte(`{"metadata": {"name": "partial-secret"}}`)

	result, err = handleSecretFromMetaManager(partialSecret)
	assert.NoError(err)
	assert.Equal("partial-secret", result.Name)
	assert.Nil(result.Data)
}
