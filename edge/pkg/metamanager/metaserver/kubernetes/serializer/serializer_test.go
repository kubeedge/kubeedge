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

package serializer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

func TestNewNegotiatedSerializer(t *testing.T) {
	assert := assert.New(t)

	serializer := NewNegotiatedSerializer()
	assert.NotNil(serializer)
	assert.IsType(WithoutConversionCodecFactory{}, serializer)

	codecFactory, ok := serializer.(WithoutConversionCodecFactory)
	assert.True(ok, "Serializer should be castable to WithoutConversionCodecFactory")
	assert.NotNil(codecFactory.creator)
	assert.NotNil(codecFactory.typer)

	supportedMediaTypes := codecFactory.SupportedMediaTypes()
	assert.Len(supportedMediaTypes, 2)
	assert.Equal("application/json", supportedMediaTypes[0].MediaType, "First media type should be application/json")
	assert.Equal("application/yaml", supportedMediaTypes[1].MediaType, "Second media type should be application/yaml")
}

func TestSupportedMediaTypes(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{}
	mediaTypes := factory.SupportedMediaTypes()

	assert.Len(mediaTypes, 2)

	jsonInfo := mediaTypes[0]
	assert.Equal("application/json", jsonInfo.MediaType)
	assert.Equal("application", jsonInfo.MediaTypeType)
	assert.Equal("json", jsonInfo.MediaTypeSubType)
	assert.True(jsonInfo.EncodesAsText)
	assert.IsType(&json.Serializer{}, jsonInfo.Serializer)
	assert.IsType(&json.Serializer{}, jsonInfo.PrettySerializer)
	assert.IsType(&json.Serializer{}, jsonInfo.StrictSerializer)
	assert.NotNil(jsonInfo.StreamSerializer)
	assert.True(jsonInfo.StreamSerializer.EncodesAsText)
	assert.IsType(&json.Serializer{}, jsonInfo.StreamSerializer.Serializer)
	assert.NotNil(jsonInfo.StreamSerializer.Framer)

	yamlInfo := mediaTypes[1]
	assert.Equal("application/yaml", yamlInfo.MediaType)
	assert.Equal("application", yamlInfo.MediaTypeType)
	assert.Equal("yaml", yamlInfo.MediaTypeSubType)
	assert.True(yamlInfo.EncodesAsText)
	assert.IsType(&json.Serializer{}, yamlInfo.Serializer)
}

func TestEncoderForVersion(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name": "test-pod",
			},
		},
	}

	gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	gv := NewWithKindGroupVersioner(gvk)

	encoder := factory.EncoderForVersion(factory.SupportedMediaTypes()[0].Serializer, gv)

	var buf bytes.Buffer
	err := encoder.Encode(obj, &buf)

	assert.NoError(err)
	assert.Contains(buf.String(), `"apiVersion":"v1"`)
	assert.Contains(buf.String(), `"kind":"Pod"`)
}

func TestDecoderToVersion(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}

	originalDecoder := factory.SupportedMediaTypes()[0].Serializer

	gv := schema.GroupVersion{Group: "test", Version: "v1"}

	resultDecoder := factory.DecoderToVersion(originalDecoder, gv)
	assert.Equal(originalDecoder, resultDecoder)

	// testing the Decoder
	testJSON := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod"}}`
	obj, _, err := resultDecoder.Decode([]byte(testJSON), nil, nil)
	assert.NoError(err)
	assert.NotNil(obj)
}

func TestIdentifier(t *testing.T) {
	assert := assert.New(t)

	gvk := schema.GroupVersionKind{Group: "test", Version: "v1", Kind: "Pod"}
	gv := NewWithKindGroupVersioner(gvk)

	encoder := &SetVersionEncoder{
		Version: gv,
		encoder: nil,
	}

	identifier := encoder.Identifier()
	expectedIdentifier := runtime.Identifier("SetVersionEncoder:test/v1, Kind=Pod")
	assert.Equal(expectedIdentifier, identifier)
}

func TestEncode(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}

	gvk := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	gv := NewWithKindGroupVersioner(gvk)

	realEncoder := factory.SupportedMediaTypes()[0].Serializer
	encoder := &SetVersionEncoder{
		Version: gv,
		encoder: realEncoder,
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name": "test-pod",
			},
		},
	}

	var buf bytes.Buffer
	err := encoder.Encode(obj, &buf)

	assert.NoError(err)

	encodedContent := buf.String()
	assert.Contains(encodedContent, `"apiVersion":"apps/v1"`)
	assert.Contains(encodedContent, `"kind":"Deployment"`)
	assert.Contains(encodedContent, `"name":"test-pod"`)
}

func TestDecode(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}

	realDecoder := factory.SupportedMediaTypes()[0].Serializer

	adapterDecoder := &AdapterDecoder{
		Decoder: realDecoder,
	}

	testObj := &unstructured.Unstructured{}
	defaultGVK := &schema.GroupVersionKind{Group: "default", Version: "v1", Kind: "DefaultKind"}

	// Test case 1: Incorrect JSON
	json := []byte(`{"apiVersion": "v1", "kind": "Pod", "metadata": {`)

	_, resultGVK, err := adapterDecoder.Decode(json, defaultGVK, testObj)
	assert.Error(err)
	assert.Nil(resultGVK)
	assert.Equal(&schema.GroupVersionKind{Group: "default", Version: "v1", Kind: "DefaultKind"}, defaultGVK, "Default GVK should remain unchanged")

	// Test case 2: Correct JSON
	correctJSON := []byte(`{"apiVersion": "v1", "kind": "Pod", "metadata": {"name": "test-pod"}}`)

	decodedObj, resultGVK, err := adapterDecoder.Decode(correctJSON, defaultGVK, testObj)
	assert.NoError(err)
	assert.NotNil(resultGVK)
	assert.Equal(&schema.GroupVersionKind{Version: "v1", Kind: "Pod"}, resultGVK, "ResultGVK should match the GVK in the JSON")

	// Check the decoded object
	unstructuredObj, ok := decodedObj.(*unstructured.Unstructured)
	assert.True(ok, "Decoded object should be of type *unstructured.Unstructured")
	assert.Equal("test-pod", unstructuredObj.GetName())

	objGVK := unstructuredObj.GetObjectKind().GroupVersionKind()
	assert.Equal(schema.GroupVersionKind{Version: "v1", Kind: "Pod"}, objGVK)
}

func TestKindForGroupVersionKinds(t *testing.T) {
	assert := assert.New(t)

	expectedGVK := schema.GroupVersionKind{Group: "test", Version: "v1", Kind: "TestKind"}
	versioner := NewWithKindGroupVersioner(expectedGVK)

	testCases := [][]schema.GroupVersionKind{
		{}, // Empty slice
		{
			{Group: "other", Version: "v2", Kind: "OtherKind"},
		},
		{
			{Group: "app", Version: "v1", Kind: "Deployment"},
			{Group: "core", Version: "v1", Kind: "Pod"},
		},
	}

	for _, inputGVK := range testCases {
		resultGVK, ok := versioner.KindForGroupVersionKinds(inputGVK)

		assert.True(ok)
		assert.Equal(expectedGVK, resultGVK)
	}
}

func TestWithKindGroupVersioner_Identifier(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name     string
		gvk      schema.GroupVersionKind
		expected string
	}{
		{
			name:     "Group, Version, and Kind",
			gvk:      schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			expected: "apps/v1, Kind=Deployment",
		},
		{
			name:     "No Group",
			gvk:      schema.GroupVersionKind{Version: "v1", Kind: "Pod"},
			expected: "/v1, Kind=Pod",
		},
		{
			name:     "No Version",
			gvk:      schema.GroupVersionKind{Group: "custom", Kind: "MyResource"},
			expected: "custom/, Kind=MyResource",
		},
		{
			name:     "No Kind",
			gvk:      schema.GroupVersionKind{Group: "custom", Version: "v1"},
			expected: "custom/v1, Kind=",
		},
		{
			name:     "Empty GVK",
			gvk:      schema.GroupVersionKind{},
			expected: "/, Kind=",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			versioner := NewWithKindGroupVersioner(tc.gvk)
			identifier := versioner.Identifier()
			assert.Equal(tc.expected, identifier)
		})
	}
}

func TestNewWithKindGroupVersioner(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name string
		gvk  schema.GroupVersionKind
	}{
		{
			name: "Standard GVK",
			gvk:  schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
		},
		{
			name: "Core API GVK",
			gvk:  schema.GroupVersionKind{Version: "v1", Kind: "Pod"},
		},
		{
			name: "Custom Resource GVK",
			gvk:  schema.GroupVersionKind{Group: "custom.example.com", Version: "v1beta1", Kind: "MyResource"},
		},
		{
			name: "Empty GVK",
			gvk:  schema.GroupVersionKind{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			versioner := NewWithKindGroupVersioner(tc.gvk)
			assert.NotNil(versioner)
			resultGVK, ok := versioner.KindForGroupVersionKinds(nil)
			assert.True(ok)
			assert.Equal(tc.gvk, resultGVK)
		})
	}
}
