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
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

const (
	namespace = "test-namespace"
	// FailedDBOperation is common Database operation fail message
	FailedDBOperation = "Failed DB Operation"
)

var errFailedDBOperation = errors.New(FailedDBOperation)

func TestNewConfigMaps(t *testing.T) {
	assert := assert.New(t)

	sender := newSend()

	cm := newConfigMaps(namespace, sender)

	assert.NotNil(cm)
	assert.Equal(namespace, cm.namespace)
	assert.Equal(sender, cm.send)
}

func TestConfigMaps_Get(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	querySetterMock := beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock

	configMapName := "test-configmap"
	expectedConfigMap := &api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	testCases := []struct {
		name      string
		respFunc  func(*model.Message) (*model.Message, error)
		stdResult *api.ConfigMap
		expectErr bool
	}{
		{
			name: "Get from MetaManager",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = expectedConfigMap
				return resp, nil
			},
			stdResult: expectedConfigMap,
			expectErr: false,
		},
		{
			name: "Error response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("test error")
			},
			stdResult: nil,
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal("test-namespace/configmap/test-configmap", message.GetResource())
				assert.Equal(model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			configMapsClient := newConfigMaps(namespace, mockSend)
			querySetterMock.EXPECT().All(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
			querySetterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySetterMock).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySetterMock).Times(1)
			configMap, err := configMapsClient.Get(configMapName)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(configMap)
			} else {
				assert.NoError(err)
				assert.Equal(test.stdResult, configMap)
			}
		})
	}
}

func TestHandleConfigMapFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name              string
		metas             []string
		expectedConfigMap *api.ConfigMap
		expectedErr       bool
	}{
		{
			name:  "Valid ConfigMap",
			metas: []string{"{\"metadata\":{\"name\":\"test-config\",\"namespace\":\"default\"},\"data\":{\"key\":\"value\"}}"},
			expectedConfigMap: &api.ConfigMap{
				Data: map[string]string{"key": "value"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
			},
			expectedErr: false,
		},
		{
			name:              "Invalid JSON",
			metas:             []string{"invalid json"},
			expectedConfigMap: nil,
			expectedErr:       true,
		},
		{
			name:              "Empty list",
			metas:             []string{},
			expectedConfigMap: nil,
			expectedErr:       true,
		},
		{
			name:              "Multiple ConfigMaps",
			metas:             []string{"", ""},
			expectedConfigMap: nil,
			expectedErr:       true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cm, err := handleConfigMapFromMetaDB(&test.metas)

			if test.expectedErr {
				assert.Error(err)
				assert.Nil(cm)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedConfigMap.Data, cm.Data)
				assert.Equal(test.expectedConfigMap.ObjectMeta.Name, cm.ObjectMeta.Name)
				assert.Equal(test.expectedConfigMap.ObjectMeta.Namespace, cm.ObjectMeta.Namespace)
			}
		})
	}
}

func TestHandleConfigMapFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name              string
		content           []byte
		expectedconfigMap *api.ConfigMap
		expectedErr       bool
	}{
		{
			name:    "Valid ConfigMap",
			content: []byte(`{"metadata":{"name":"test-config","namespace":"default"},"data":{"key":"value"}}`),
			expectedconfigMap: &api.ConfigMap{
				Data: map[string]string{"key": "value"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
			},
			expectedErr: false,
		},
		{
			name:              "Invalid JSON",
			content:           []byte(`{"invalid json"`),
			expectedconfigMap: nil,
			expectedErr:       true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cm, err := handleConfigMapFromMetaManager(test.content)

			if test.expectedErr {
				assert.Error(err)
				assert.Nil(cm)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedconfigMap.Data, cm.Data)
				assert.Equal(test.expectedconfigMap.ObjectMeta.Name, cm.ObjectMeta.Name)
				assert.Equal(test.expectedconfigMap.ObjectMeta.Namespace, cm.ObjectMeta.Namespace)
			}
		})
	}
}
