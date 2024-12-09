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
	"testing"

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

	// Test case 1: Valid PersistentVolume JSON in list
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
}

func TestHandlePersistentVolumeFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid PersistentVolume JSON
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
