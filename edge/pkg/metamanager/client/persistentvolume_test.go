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

func TestNewPersistentVolumes(t *testing.T) {
	assert := assert.New(t)

	s := newSend()
	pv := newPersistentVolumes(s)

	assert.NotNil(pv)
	assert.IsType(&send{}, pv.send)
}

func TestHandlePersistentVolumeFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	pv := &api.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: api.PersistentVolumeSpec{
			Capacity: api.ResourceList{
				api.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}
	pvJSON, _ := json.Marshal(pv)
	validList := []string{string(pvJSON)}
	validContent, _ := json.Marshal(validList)

	result, err := handlePersistentVolumeFromMetaDB(validContent)
	assert.NoError(err)
	assert.Equal(pv, result)

	emptyList := []string{}
	emptyContent, _ := json.Marshal(emptyList)

	result, err = handlePersistentVolumeFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "persistentvolume length from meta db is 0")

	invalidList := []string{`{"invalid": json}`}
	invalidContent, _ := json.Marshal(invalidList)

	result, err = handlePersistentVolumeFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolume from db failed")
}

func TestHandlePersistentVolumeFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	pv := &api.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: api.PersistentVolumeSpec{
			Capacity: api.ResourceList{
				api.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}
	content, _ := json.Marshal(pv)

	result, err := handlePersistentVolumeFromMetaManager(content)
	assert.NoError(err)
	assert.Equal(pv, result)

	emptyContent := []byte("{}")

	result, err = handlePersistentVolumeFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.Equal(&api.PersistentVolume{}, result)

	invalidContent := []byte(`{"invalid": json}`)

	result, err = handlePersistentVolumeFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolume failed")
}

func TestPersistentVolumes_Create(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	pv := newPersistentVolumes(s)

	input := &api.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: api.PersistentVolumeSpec{
			Capacity: api.ResourceList{
				api.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}

	result, err := pv.Create(input)

	assert.Nil(result)
	assert.NoError(err)
}

func TestPersistentVolumes_Update(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	pv := newPersistentVolumes(s)

	input := &api.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: api.PersistentVolumeSpec{
			Capacity: api.ResourceList{
				api.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}

	err := pv.Update(input)
	assert.NoError(err)
}

func TestPersistentVolumes_Delete(t *testing.T) {
	assert := assert.New(t)

	s := newMockSend()
	pv := newPersistentVolumes(s)

	err := pv.Delete("test-pv")
	assert.NoError(err)
}

func TestPersistentVolumes_Get(t *testing.T) {
	testCases := []struct {
		name        string
		pvName      string
		mockSetup   func(*mockSend)
		expectError bool
		expectPV    *api.PersistentVolume
	}{
		{
			name:   "successful get from MetaDB",
			pvName: "test-pv",
			mockSetup: func(m *mockSend) {
				m.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					resp := model.NewMessage(msg.GetID())
					resp.Router.Operation = model.ResponseOperation
					resp.Router.Source = modules.MetaManagerModuleName

					pv := &api.PersistentVolume{
						ObjectMeta: metav1.ObjectMeta{Name: "test-pv"},
						Spec: api.PersistentVolumeSpec{
							Capacity: api.ResourceList{
								api.ResourceStorage: resource.MustParse("5Gi"),
							},
						},
					}
					pvJSON, _ := json.Marshal(pv)
					content, _ := json.Marshal([]string{string(pvJSON)})
					resp.Content = content
					return resp, nil
				}
			},
			expectError: false,
			expectPV: &api.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pv"},
				Spec: api.PersistentVolumeSpec{
					Capacity: api.ResourceList{
						api.ResourceStorage: resource.MustParse("5Gi"),
					},
				},
			},
		},
		{
			name:   "error from SendSync",
			pvName: "test-pv",
			mockSetup: func(m *mockSend) {
				m.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
					return nil, fmt.Errorf("send sync error")
				}
			},
			expectError: true,
			expectPV:    nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			mock := newMockSend()
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			pv := newPersistentVolumes(mock)
			result, err := pv.Get(tt.pvName, metav1.GetOptions{})

			if tt.expectError {
				assert.Error(err)
				assert.Nil(result)
			} else {
				assert.NoError(err)
				assert.Equal(tt.expectPV, result)
			}
		})
	}
}
