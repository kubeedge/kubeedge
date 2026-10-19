package storage

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDecodeAndConvertCore(t *testing.T) {
	assert := assert.New(t)

	// Test core type (Pod)
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
	}

	podJSON, err := json.Marshal(pod)
	assert.NoError(err)

	// Should successfully decode core type
	result, err := DecodeAndConvert(podJSON, "")
	assert.NoError(err)

	resultPod, ok := result.(*corev1.Pod)
	assert.True(ok)
	assert.Equal("test-pod", resultPod.Name)
	assert.Equal("nginx", resultPod.Spec.Containers[0].Name)
}

func TestDecodeAndConvertAPIExtensions(t *testing.T) {
	assert := assert.New(t)

	// Test apiextensions type (CustomResourceDefinition)
	crd := &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "widgets.example.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "widgets",
				Singular: "widget",
				Kind:     "Widget",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
						},
					},
				},
			},
		},
	}

	crdJSON, err := json.Marshal(crd)
	assert.NoError(err)

	// Should successfully decode apiextensions type
	result, err := DecodeAndConvert(crdJSON, "apiextensions.k8s.io")
	assert.NoError(err)

	resultCRD, ok := result.(*apiextensionsv1.CustomResourceDefinition)
	assert.True(ok)
	assert.Equal("widgets.example.com", resultCRD.Name)
	assert.Equal("example.com", resultCRD.Spec.Group)
}

func TestDecodeAndConvertUnregisteredCRD(t *testing.T) {
	assert := assert.New(t)

	// Test unregistered CRD type
	customObj := map[string]interface{}{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"metadata": map[string]interface{}{
			"name": "test-widget",
		},
		"spec": map[string]interface{}{
			"size":   "large",
			"color":  "blue",
			"weight": 100,
		},
	}

	customJSON, err := json.Marshal(customObj)
	assert.NoError(err)

	// Should return runtime.Unknown for unregistered type
	result, err := DecodeAndConvert(customJSON, "example.com")
	assert.NoError(err)

	unknown, ok := result.(*runtime.Unknown)
	assert.True(ok)
	assert.Equal(runtime.ContentTypeJSON, unknown.ContentType)

	// Verify the raw JSON is preserved
	var decoded map[string]interface{}
	err = json.Unmarshal(unknown.Raw, &decoded)
	assert.NoError(err)
	assert.Equal("test-widget", decoded["metadata"].(map[string]interface{})["name"])
}

func TestDecodeAndConvertInvalidJSON(t *testing.T) {
	assert := assert.New(t)

	// Should return error for invalid JSON
	invalidJSON := []byte(`{"invalid": json`)
	result, err := DecodeAndConvert(invalidJSON, "")
	assert.Error(err)
	assert.True(strings.Contains(err.Error(), "failed to decode"))
	assert.Nil(result)
}

func TestDecodeAndConvertEmptyBody(t *testing.T) {
	assert := assert.New(t)

	// Should handle empty body gracefully
	result, err := DecodeAndConvert([]byte{}, "")
	assert.Error(err)
	assert.True(strings.Contains(err.Error(), "failed to decode"))
	assert.Nil(result)
}

func TestDecodeAndConvertList(t *testing.T) {
	assert := assert.New(t)

	// Test list type
	list := &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PodList",
		},
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod2",
				},
			},
		},
	}

	listJSON, err := json.Marshal(list)
	assert.NoError(err)

	// Should successfully decode list type
	result, err := DecodeAndConvert(listJSON, "")
	assert.NoError(err)

	resultList, ok := result.(*corev1.PodList)
	assert.True(ok)
	assert.Len(resultList.Items, 2)
	assert.Equal("pod1", resultList.Items[0].Name)
	assert.Equal("pod2", resultList.Items[1].Name)
}
