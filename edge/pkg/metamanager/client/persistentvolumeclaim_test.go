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

	// Test case 1: Valid PersistentVolumeClaim JSON
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

	// Test case 2: Empty list
	emptyContent, _ := json.Marshal([]string{})

	result, err = handlePersistentVolumeClaimFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "persistentvolumeclaim length from meta db is 0")

	// Test case 3: Multiple PVCs in list
	multiplePVCs, _ := json.Marshal([]string{string(pvcJSON), string(pvcJSON)})

	result, err = handlePersistentVolumeClaimFromMetaDB(multiplePVCs)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "persistentvolumeclaim length from meta db is 2")

	// Test case 4: Invalid JSON in list
	invalidJSON := []byte(`["{invalid json}"]`)

	result, err = handlePersistentVolumeClaimFromMetaDB(invalidJSON)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolumeclaim from db failed")

	// Test case 5: Invalid outer JSON
	invalidOuterJSON := []byte(`{"not": "a list"}`)

	result, err = handlePersistentVolumeClaimFromMetaDB(invalidOuterJSON)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolumeclaim list from db failed")
}

func TestHandlePersistentVolumeClaimFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid PersistentVolumeClaim JSON
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

	// Test case 2: Empty JSON
	emptyContent := []byte("{}")

	result, err = handlePersistentVolumeClaimFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.Equal(&api.PersistentVolumeClaim{}, result)

	// Test case 3: Invalid JSON
	invalidContent := []byte(`{"invalid": json}`)

	result, err = handlePersistentVolumeClaimFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to persistentvolumeclaim failed")
}
