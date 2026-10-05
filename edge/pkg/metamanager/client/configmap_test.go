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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestConfigMapsCreate(t *testing.T) {
	cm := &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	mockSend := &mockSendInterface{}
	mockMeta := &MockMetaService{}
	configMaps := NewConfigMapsWithMetaService("default", mockSend, mockMeta)
	result, err := configMaps.Create(cm)

	assert.Nil(t, err)
	assert.Nil(t, result)
}

func TestConfigMapsUpdate(t *testing.T) {
	cm := &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "updated-value",
		},
	}

	mockSend := &mockSendInterface{}
	mockMeta := &MockMetaService{}
	configMaps := NewConfigMapsWithMetaService("default", mockSend, mockMeta)
	err := configMaps.Update(cm)

	assert.Nil(t, err)
}

func TestConfigMapsDelete(t *testing.T) {
	mockSend := &mockSendInterface{}
	mockMeta := &MockMetaService{}
	configMaps := NewConfigMapsWithMetaService("default", mockSend, mockMeta)
	err := configMaps.Delete("test-cm")

	assert.Nil(t, err)
}

func TestConfigMapsGetFromMetaDB_Success(t *testing.T) {
	testCM := &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	cmData, _ := json.Marshal(testCM)
	cmList := []string{string(cmData)}

	mockMeta := &MockMetaService{
		QueryMetaFunc: func(key, value string) (*[]string, error) {
			return &cmList, nil
		},
	}

	mockSend := &mockSendInterface{}
	configMaps := NewConfigMapsWithMetaService("default", mockSend, mockMeta)
	result, err := configMaps.Get("test-cm")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-cm", result.Name)
	assert.Equal(t, "default", result.Namespace)
	assert.Equal(t, "value", result.Data["key"])
}

func TestConfigMapsGetFromMetaDB_QueryError(t *testing.T) {
	mockMeta := &MockMetaService{
		QueryMetaFunc: func(key, value string) (*[]string, error) {
			return nil, fmt.Errorf("query failed")
		},
	}

	mockSend := &mockSendInterface{
		sendSyncFunc: func(msg *model.Message) (*model.Message, error) {
			return nil, fmt.Errorf("remote get failed")
		},
	}

	configMaps := NewConfigMapsWithMetaService("default", mockSend, mockMeta)
	result, err := configMaps.Get("test-cm")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "remote get failed")
}

func TestConfigMapsGetFromMetaDB_Empty_FallbackToRemote(t *testing.T) {
	testCM := &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	cmData, _ := json.Marshal(testCM)

	mockMeta := &MockMetaService{
		QueryMetaFunc: func(key, value string) (*[]string, error) {
			return &[]string{}, nil
		},
	}

	mockSend := &mockSendInterface{
		sendSyncFunc: func(msg *model.Message) (*model.Message, error) {
			return &model.Message{
				Content: cmData,
			}, nil
		},
	}

	configMaps := NewConfigMapsWithMetaService("default", mockSend, mockMeta)
	result, err := configMaps.Get("test-cm")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-cm", result.Name)
}

func TestHandleConfigMapFromMetaDB_Success(t *testing.T) {
	testCM := &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	cmData, err := json.Marshal(testCM)
	assert.NoError(t, err)

	lists := []string{string(cmData)}
	result, err := handleConfigMapFromMetaDB(&lists)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testCM.Name, result.Name)
	assert.Equal(t, testCM.Namespace, result.Namespace)
	assert.Equal(t, testCM.Data, result.Data)
}

func TestHandleConfigMapFromMetaDB_EmptyList(t *testing.T) {
	lists := []string{}
	result, err := handleConfigMapFromMetaDB(&lists)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "ConfigMap length from meta db is 0")
}

func TestHandleConfigMapFromMetaDB_MultipleItems(t *testing.T) {
	lists := []string{"cm1", "cm2"}
	result, err := handleConfigMapFromMetaDB(&lists)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "ConfigMap length from meta db is 2")
}

func TestHandleConfigMapFromMetaDB_UnmarshalError(t *testing.T) {
	lists := []string{"invalid-json"}
	result, err := handleConfigMapFromMetaDB(&lists)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unmarshal message to ConfigMap from db failed")
}

func TestHandleConfigMapFromMetaManager_Success(t *testing.T) {
	testCM := &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"config":  "data",
			"setting": "value",
		},
	}

	content, err := json.Marshal(testCM)
	assert.NoError(t, err)

	result, err := handleConfigMapFromMetaManager(content)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testCM.Name, result.Name)
	assert.Equal(t, len(testCM.Data), len(result.Data))
	assert.Equal(t, "data", result.Data["config"])
}

func TestHandleConfigMapFromMetaManager_UnmarshalError(t *testing.T) {
	invalidContent := []byte("invalid-json")

	result, err := handleConfigMapFromMetaManager(invalidContent)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unmarshal message to ConfigMap failed")
}

func TestHandleConfigMapFromMetaManager_EmptyContent(t *testing.T) {
	content := []byte("{}")

	result, err := handleConfigMapFromMetaManager(content)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Name)
}

func TestConfigMapsInterface(t *testing.T) {
	mockSend := newMockSend()
	configMapsClient := newConfigMaps(testNamespace, mockSend)

	var _ ConfigMapsInterface = configMapsClient
	assert.NotNil(t, configMapsClient)
}
