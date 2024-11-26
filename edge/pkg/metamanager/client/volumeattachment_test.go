package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	api "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewVolumeAttachments(t *testing.T) {
	assert := assert.New(t)
	sendInterface := newSend()

	va := newVolumeAttachments(namespace, sendInterface)
	assert.NotNil(va)
	assert.Equal(namespace, va.namespace)
	assert.Equal(sendInterface, va.send)
}

func TestHandleVolumeAttachmentFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid VolumeAttachment JSON in array
	validVA := &api.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: api.VolumeAttachmentSpec{
			Attacher: "test-attacher",
			Source: api.VolumeAttachmentSource{
				PersistentVolumeName: func() *string {
					s := "test-volume"
					return &s
				}(),
			},
			NodeName: "test-node",
		},
	}
	vaJSON, _ := json.Marshal(validVA)
	validContent, _ := json.Marshal([]string{string(vaJSON)})

	va, err := handleVolumeAttachmentFromMetaDB(validContent)
	assert.NoError(err)
	assert.NotNil(va)
	assert.Equal(validVA.Name, va.Name)
	assert.Equal(validVA.Spec.Attacher, va.Spec.Attacher)
	assert.Equal(validVA.Spec.NodeName, va.Spec.NodeName)
	assert.Equal(*validVA.Spec.Source.PersistentVolumeName, *va.Spec.Source.PersistentVolumeName)

	// Test case 2: Invalid JSON
	invalidContent := []byte("invalid json")

	va, err = handleVolumeAttachmentFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(va)
	assert.Contains(err.Error(), "unmarshal message to volumeattachment list from db failed")

	// Test case 3: Empty array
	emptyContent, _ := json.Marshal([]string{})

	va, err = handleVolumeAttachmentFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(va)
	assert.Contains(err.Error(), "volumeattachment length from meta db is 0")

	// Test case 4: Array with multiple elements
	multipleContent, _ := json.Marshal([]string{"{}", "{}"})

	va, err = handleVolumeAttachmentFromMetaDB(multipleContent)
	assert.Error(err)
	assert.Nil(va)
	assert.Contains(err.Error(), "volumeattachment length from meta db is 2")
}

func TestHandleVolumeAttachmentFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid VolumeAttachment JSON
	validVA := &api.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-va",
		},
		Spec: api.VolumeAttachmentSpec{
			Attacher: "test-attacher",
			Source: api.VolumeAttachmentSource{
				PersistentVolumeName: func() *string {
					s := "test-volume"
					return &s
				}(),
			},
			NodeName: "test-node",
		},
	}

	validContent, _ := json.Marshal(validVA)

	va, err := handleVolumeAttachmentFromMetaManager(validContent)
	assert.NoError(err)
	assert.NotNil(va)
	assert.Equal(validVA.Name, va.Name)
	assert.Equal(validVA.Spec.Attacher, va.Spec.Attacher)
	assert.Equal(validVA.Spec.NodeName, va.Spec.NodeName)
	assert.Equal(*validVA.Spec.Source.PersistentVolumeName, *va.Spec.Source.PersistentVolumeName)

	// Test case 2: Invalid JSON
	invalidContent := []byte("invalid json")

	va, err = handleVolumeAttachmentFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(va)
	assert.Contains(err.Error(), "unmarshal message to volumeattachment failed")

	// Test case 3: Empty JSON object
	emptyContent := []byte("{}")

	va, err = handleVolumeAttachmentFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.NotNil(va)
	assert.Empty(va.Name)
	assert.Empty(va.Spec.Attacher)
	assert.Empty(va.Spec.NodeName)
	assert.Nil(va.Spec.Source.PersistentVolumeName)
}
