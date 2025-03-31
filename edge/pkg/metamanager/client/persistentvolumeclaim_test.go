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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// Common test constants used throughout the tests
const (
	testPVCName = "test-pvc"
)

// setupTest creates common test objects and returns them
func setupTest(t *testing.T) (*assert.Assertions, SendInterface, *persistentvolumeclaims, *gomonkey.Patches) {
	a := assert.New(t)
	s := newSend()
	pvcClient := newPersistentVolumeClaims(testNamespace, s)
	patches := gomonkey.NewPatches()
	t.Cleanup(func() {
		patches.Reset()
	})
	return a, s, pvcClient, patches
}

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
			Name:      testPVCName,
			Namespace: testNamespace,
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
			Name:      testPVCName,
			Namespace: testNamespace,
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

func TestPersistentVolumeClaimsStubMethods(t *testing.T) {
	assert := assert.New(t)

	s := newSend()
	pvcClient := newPersistentVolumeClaims(testNamespace, s)

	testPVC := &api.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPVCName,
			Namespace: testNamespace,
		},
	}

	result, err := pvcClient.Create(testPVC)
	assert.Nil(result)
	assert.NoError(err)

	err = pvcClient.Update(testPVC)
	assert.NoError(err)

	err = pvcClient.Delete(testPVCName)
	assert.NoError(err)
}

func TestPersistentVolumeClaimsGet(t *testing.T) {
	t.Run("SendSync Error", func(t *testing.T) {
		assert, s, pvcClient, patches := setupTest(t)

		patches.ApplyMethod(s, "SendSync",
			func(_ *send, _ *model.Message) (*model.Message, error) {
				return nil, errors.New("send sync error")
			})

		result, err := pvcClient.Get(testPVCName, metav1.GetOptions{})

		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "get persistentvolumeclaim from metaManager failed")
	})

	t.Run("GetContentData Error", func(t *testing.T) {
		assert, s, pvcClient, patches := setupTest(t)
		mockMsg := &model.Message{}

		patches.ApplyMethod(s, "SendSync",
			func(_ *send, _ *model.Message) (*model.Message, error) {
				return mockMsg, nil
			})

		patches.ApplyMethod(mockMsg, "GetContentData",
			func(_ *model.Message) ([]byte, error) {
				return nil, errors.New("get content data error")
			})

		result, err := pvcClient.Get(testPVCName, metav1.GetOptions{})

		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "parse message to persistentvolumeclaim failed")
	})

	t.Run("MetaManager Handler Error", func(t *testing.T) {
		assert, s, pvcClient, patches := setupTest(t)
		mockMsg := &model.Message{}

		patches.ApplyMethod(s, "SendSync",
			func(_ *send, _ *model.Message) (*model.Message, error) {
				return mockMsg, nil
			})

		patches.ApplyMethod(mockMsg, "GetOperation",
			func(_ *model.Message) string {
				return "OtherOperation"
			})

		patches.ApplyMethod(mockMsg, "GetSource",
			func(_ *model.Message) string {
				return modules.MetaManagerModuleName
			})

		patches.ApplyMethod(mockMsg, "GetContentData",
			func(_ *model.Message) ([]byte, error) {
				return []byte(`{"invalid json`), nil
			})

		result, err := pvcClient.Get(testPVCName, metav1.GetOptions{})

		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "unmarshal message to persistentvolumeclaim failed")
	})

	t.Run("Test BuildMsg Parameters", func(t *testing.T) {
		assert, s, pvcClient, patches := setupTest(t)

		expectedResource := fmt.Sprintf("%s/%s/%s", testNamespace, "persistentvolumeclaim", testPVCName)

		var capturedGroup, capturedResource, capturedSource, capturedDest string
		var capturedOperation string
		var capturedContent interface{}

		patches.ApplyFunc(message.BuildMsg,
			func(group, source, dest, resource string, operation string, content interface{}) *model.Message {
				capturedGroup = group
				capturedSource = source
				capturedDest = dest
				capturedResource = resource
				capturedOperation = operation
				capturedContent = content

				return &model.Message{}
			})

		patches.ApplyMethod(s, "SendSync",
			func(_ *send, _ *model.Message) (*model.Message, error) {
				return nil, errors.New("some error")
			})

		result, err := pvcClient.Get(testPVCName, metav1.GetOptions{})

		assert.Error(err)
		assert.Nil(result)
		assert.Equal(modules.MetaGroup, capturedGroup)
		assert.Equal("", capturedSource)
		assert.Equal(modules.EdgedModuleName, capturedDest)
		assert.Equal(expectedResource, capturedResource)
		assert.Equal(model.QueryOperation, capturedOperation)
		assert.Nil(capturedContent)
	})
}
