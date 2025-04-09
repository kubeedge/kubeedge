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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
)

const (
	testPVName = "test-pv"
)

func createTestPersistentVolume() *api.PersistentVolume {
	return &api.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: testPVName,
		},
		Spec: api.PersistentVolumeSpec{
			Capacity: api.ResourceList{
				api.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}
}

func TestNewPersistentVolumes(t *testing.T) {
	assert := assert.New(t)

	s := newSend()
	pv := newPersistentVolumes(s)

	assert.NotNil(pv)
	assert.IsType(&send{}, pv.send)
}

func TestPersistentVolume_Create(t *testing.T) {
	assert := assert.New(t)

	inputPV := createTestPersistentVolume()

	pvClient := newPersistentVolumes(nil)
	createdPV, err := pvClient.Create(inputPV)

	assert.Nil(err, "Create method should return nil error")
	assert.Nil(createdPV, "Create method should return nil PV")
}

func TestPersistentVolume_Update(t *testing.T) {
	assert := assert.New(t)

	inputPV := createTestPersistentVolume()

	pvClient := newPersistentVolumes(nil)
	err := pvClient.Update(inputPV)

	assert.Nil(err, "Update method should return nil error")
}

func TestPersistentVolume_Delete(t *testing.T) {
	assert := assert.New(t)

	pvClient := newPersistentVolumes(nil)
	err := pvClient.Delete(testPVName)

	assert.Nil(err, "Delete method should return nil error")
}

func TestPersistentVolume_Get(t *testing.T) {
	assert := assert.New(t)

	expectedPV := createTestPersistentVolume()
	pvJSON, _ := json.Marshal(expectedPV)
	metaDBList := []string{string(pvJSON)}
	metaDBListJSON, _ := json.Marshal(metaDBList)

	options := metav1.GetOptions{}
	resource := fmt.Sprintf("%s/%s/%s", v2.NullNamespace, "persistentvolume", testPVName)

	testCases := []struct {
		name        string
		respFunc    func(*model.Message) (*model.Message, error)
		expectedPV  *api.PersistentVolume
		expectErr   bool
		errContains string
	}{
		{
			name: "Get PV from MetaManager",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Source = "other-module"
				resp.Content = pvJSON
				return resp, nil
			},
			expectedPV: expectedPV,
			expectErr:  false,
		},
		{
			name: "Get PV from MetaDB",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Operation = model.ResponseOperation
				resp.Router.Source = modules.MetaManagerModuleName
				resp.Content = metaDBListJSON
				return resp, nil
			},
			expectedPV: expectedPV,
			expectErr:  false,
		},
		{
			name: "SendSync Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("send sync error")
			},
			expectedPV:  nil,
			expectErr:   true,
			errContains: "get persistentvolume from metaManager failed",
		},
		{
			name: "Content Unmarshal Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = 123
				return resp, nil
			},
			expectedPV:  nil,
			expectErr:   true,
			errContains: "unmarshal message to persistentvolume failed",
		},
		{
			name: "MetaDB PV Unmarshal List Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Operation = model.ResponseOperation
				resp.Router.Source = modules.MetaManagerModuleName
				resp.Content = []byte(`{"invalid": json}`)
				return resp, nil
			},
			expectedPV:  nil,
			expectErr:   true,
			errContains: "unmarshal message to persistentvolume list from db failed",
		},
		{
			name: "MetaManager PV Unmarshal Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = []byte(`{"invalid": json}`)
				return resp, nil
			},
			expectedPV:  nil,
			expectErr:   true,
			errContains: "unmarshal message to persistentvolume failed",
		},
		{
			name: "MetaDB with multiple PVs",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Operation = model.ResponseOperation
				resp.Router.Source = modules.MetaManagerModuleName
				multipleList := []string{string(pvJSON), string(pvJSON)}
				multipleListJSON, _ := json.Marshal(multipleList)
				resp.Content = multipleListJSON
				return resp, nil
			},
			expectedPV:  nil,
			expectErr:   true,
			errContains: "persistentvolume length from meta db is 2",
		},
		{
			name: "MetaDB with invalid PV JSON",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Operation = model.ResponseOperation
				resp.Router.Source = modules.MetaManagerModuleName
				invalidList := []string{`{"invalid": json}`}
				invalidListJSON, _ := json.Marshal(invalidList)
				resp.Content = invalidListJSON
				return resp, nil
			},
			expectedPV:  nil,
			expectErr:   true,
			errContains: "unmarshal message to persistentvolume from db failed",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal(resource, message.GetResource())
				assert.Equal(model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			pvClient := newPersistentVolumes(mockSend)
			pv, err := pvClient.Get(testPVName, options)

			if test.expectErr {
				assert.Error(err)
				if test.errContains != "" {
					assert.Contains(err.Error(), test.errContains)
				}
				assert.Nil(pv)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedPV, pv)
			}
		})
	}
}

func TestHandlePersistentVolumeFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid PersistentVolume JSON in list
	pv := createTestPersistentVolume()
	pvJSON, _ := json.Marshal(pv)
	validList := []string{string(pvJSON)}
	validContent, _ := json.Marshal(validList)

	result, err := handlePersistentVolumeFromMetaDB(validContent)
	assert.NoError(err)
	assert.Equal(pv, result)

	// Test case 2: Empty list
	emptyList := []string{}
	emptyContent, _ := json.Marshal(emptyList)

	result, err = handlePersistentVolumeFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "persistentvolume length from meta db is 0")

	// Test case 3: Invalid JSON in list
	invalidList := []string{`{"invalid": json}`}
	invalidContent, _ := json.Marshal(invalidList)

	result, err = handlePersistentVolumeFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolume from db failed")

	multipleList := []string{string(pvJSON), string(pvJSON)}
	multipleContent, _ := json.Marshal(multipleList)

	result, err = handlePersistentVolumeFromMetaDB(multipleContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "persistentvolume length from meta db is 2")

	invalidContent = []byte(`{"not": "a list"}`)

	result, err = handlePersistentVolumeFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolume list from db failed")
}

func TestHandlePersistentVolumeFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid PersistentVolume JSON
	pv := createTestPersistentVolume()
	content, _ := json.Marshal(pv)

	result, err := handlePersistentVolumeFromMetaManager(content)
	assert.NoError(err)
	assert.Equal(pv, result)

	// Test case 2: Empty JSON
	emptyContent := []byte("{}")

	result, err = handlePersistentVolumeFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.Equal(&api.PersistentVolume{}, result)

	// Test case 3: Invalid JSON
	invalidContent := []byte(`{"invalid": json}`)

	result, err = handlePersistentVolumeFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolume failed")
}
