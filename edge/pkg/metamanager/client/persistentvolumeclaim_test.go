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

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/stretchr/testify/assert"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewPersistentVolumeClaims(t *testing.T) {
	assert := assert.New(t)

	s := newSend()
	pvc := newPersistentVolumeClaims(namespace, s)

	assert.NotNil(pvc)
	assert.Equal(namespace, pvc.namespace)
	assert.IsType(&send{}, pvc.send)
}

func TestHandlePersistentVolumeClaimFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	pvc := &api.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
		Spec: api.PersistentVolumeClaimSpec{
			AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteOnce},
			Resources: api.VolumeResourceRequirements{
				Requests: api.ResourceList{
					api.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	pvcJSON, _ := json.Marshal(pvc)
	content, _ := json.Marshal([]string{string(pvcJSON)})

	result, err := handlePersistentVolumeClaimFromMetaDB(content)
	assert.NoError(err)
	assert.Equal(pvc, result)

	emptyContent, _ := json.Marshal([]string{})

	result, err = handlePersistentVolumeClaimFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "persistentvolumeclaim length from meta db is 0")

	multiplePVCs, _ := json.Marshal([]string{string(pvcJSON), string(pvcJSON)})

	result, err = handlePersistentVolumeClaimFromMetaDB(multiplePVCs)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "persistentvolumeclaim length from meta db is 2")

	invalidJSON := []byte(`["{invalid json}"]`)

	result, err = handlePersistentVolumeClaimFromMetaDB(invalidJSON)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolumeclaim from db failed")

	invalidOuterJSON := []byte(`{"not": "a list"}`)

	result, err = handlePersistentVolumeClaimFromMetaDB(invalidOuterJSON)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolumeclaim list from db failed")
}

func TestHandlePersistentVolumeClaimFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	pvc := &api.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
		Spec: api.PersistentVolumeClaimSpec{
			AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteOnce},
			Resources: api.VolumeResourceRequirements{
				Requests: api.ResourceList{
					api.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	content, _ := json.Marshal(pvc)

	result, err := handlePersistentVolumeClaimFromMetaManager(content)
	assert.NoError(err)
	assert.Equal(pvc, result)

	emptyContent := []byte("{}")

	result, err = handlePersistentVolumeClaimFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.Equal(&api.PersistentVolumeClaim{}, result)

	invalidContent := []byte(`{"invalid": json}`)

	result, err = handlePersistentVolumeClaimFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolumeclaim failed")
}

func TestPersistentVolumeClaims_Create(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	pvc := newPersistentVolumeClaims(namespace, s)

	input := &api.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: namespace,
		},
		Spec: api.PersistentVolumeClaimSpec{
			AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteOnce},
			Resources: api.VolumeResourceRequirements{
				Requests: api.ResourceList{
					api.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	result, err := pvc.Create(input)

	assert.Nil(result)
	assert.NoError(err)
}

func TestPersistentVolumeClaims_Update(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	pvc := newPersistentVolumeClaims(namespace, s)

	input := &api.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: namespace,
		},
	}

	err := pvc.Update(input)

	assert.NoError(err)
}

func TestPersistentVolumeClaims_Delete(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	pvc := newPersistentVolumeClaims(namespace, s)

	err := pvc.Delete("test-pvc")

	assert.NoError(err)
}

func TestPersistentVolumeClaims_Get(t *testing.T) {
	testCases := []struct {
		name        string
		pvcName     string
		mockSetup   func(*mockSend)
		expectError bool
		expectPVC   *api.PersistentVolumeClaim
	}{
		{
			name:    "successful get from MetaDB",
			pvcName: "test-pvc",
			mockSetup: func(m *mockSend) {
				m.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					resp := model.NewMessage(msg.GetID())
					resp.Router.Operation = model.ResponseOperation
					resp.Router.Source = modules.MetaManagerModuleName

					pvc := &api.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-pvc",
							Namespace: namespace,
						},
					}
					pvcJSON, _ := json.Marshal(pvc)
					content, _ := json.Marshal([]string{string(pvcJSON)})
					resp.Content = content
					return resp, nil
				}
			},
			expectError: false,
			expectPVC: &api.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: namespace,
				},
			},
		},
		{
			name:    "error from SendSync",
			pvcName: "test-pvc",
			mockSetup: func(m *mockSend) {
				m.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					return nil, fmt.Errorf("send sync error")
				}
			},
			expectError: true,
			expectPVC:   nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			mock := newMockSend()
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			pvc := newPersistentVolumeClaims(namespace, mock)
			result, err := pvc.Get(tt.pvcName, metav1.GetOptions{})

			if tt.expectError {
				assert.Error(err)
				assert.Nil(result)
			} else {
				assert.NoError(err)
				assert.Equal(tt.expectPVC, result)
			}
		})
	}
}
