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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
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
	assert.Len(supportedMediaTypes, 3)
	assert.Equal("application/json", supportedMediaTypes[0].MediaType, "First media type should be application/json")
	assert.Equal("application/yaml", supportedMediaTypes[1].MediaType, "Second media type should be application/yaml")
	assert.Equal("application/vnd.kubernetes.protobuf", supportedMediaTypes[2].MediaType, "Third media type should be protobuf")
}

func TestSupportedMediaTypes(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{}
	mediaTypes := factory.SupportedMediaTypes()

	assert.Len(mediaTypes, 3)

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

	// protobuf info
	protoInfo := mediaTypes[2]
	assert.Equal("application/vnd.kubernetes.protobuf", protoInfo.MediaType)
	assert.Equal("application", protoInfo.MediaTypeType)
	assert.Equal("vnd.kubernetes.protobuf", protoInfo.MediaTypeSubType)
	assert.False(protoInfo.EncodesAsText)
	// verify serializer types: protobuf serializer is used for protobuf media type
	assert.IsType(&protobuf.Serializer{}, protoInfo.Serializer)
	assert.IsType(&protobuf.Serializer{}, protoInfo.StrictSerializer)
	// stream serializer
	assert.NotNil(protoInfo.StreamSerializer)
	assert.False(protoInfo.StreamSerializer.EncodesAsText)
	assert.IsType(&protobuf.RawSerializer{}, protoInfo.StreamSerializer.Serializer)
	// framer should be the length-delimited framer used by protobuf stream serialization
	assert.Equal(protobuf.LengthDelimitedFramer, protoInfo.StreamSerializer.Framer)
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

func TestEncodePodListJSON(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}

	// Build a simple PodList unstructured
	podList := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "PodList",
			"items": []interface{}{
				map[string]interface{}{"metadata": map[string]interface{}{"name": "pod1"}},
				map[string]interface{}{"metadata": map[string]interface{}{"name": "pod2"}},
			},
		},
	}

	// Use the JSON serializer (first supported media type)
	serializer := factory.SupportedMediaTypes()[0].Serializer

	var buf bytes.Buffer
	err := serializer.Encode(podList, &buf)
	assert.NoError(err)

	out := buf.String()
	assert.Contains(out, "PodList")
	assert.Contains(out, "pod1")
	assert.Contains(out, "pod2")
}

func TestEncodeDecodePodListProtobufUnstructuredError(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}

	// Build a simple PodList unstructured
	podList := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "PodList",
			"items": []interface{}{
				map[string]interface{}{"metadata": map[string]interface{}{"name": "pod1"}},
				map[string]interface{}{"metadata": map[string]interface{}{"name": "pod2"}},
			},
		},
	}

	// Use the protobuf serializer (third supported media type)
	protoSerializer := factory.SupportedMediaTypes()[2].Serializer

	var buf bytes.Buffer
	// Encoding unstructured with the generic protobuf serializer is expected to fail
	// since unstructured objects are not protobuf-serializable.
	err := protoSerializer.Encode(podList, &buf)
	assert.Error(err)
}

func TestEncodeDecodePodListProtobuf(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}

	podList := &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodList",
			APIVersion: "v1",
		},
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "pod1"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "pod2"},
			},
		},
	}

	// Use the protobuf serializer (third supported media type)
	protoSerializer := factory.SupportedMediaTypes()[2].Serializer

	var buf bytes.Buffer
	// Encoding unstructured with the generic protobuf serializer is expected to fail
	// since unstructured objects are not protobuf-serializable.
	err := protoSerializer.Encode(podList, &buf)
	assert.Nil(err)
}

func TestProtobufFramerRoundTrip(t *testing.T) {
	assert := assert.New(t)

	var buf bytes.Buffer
	// Write two raw frames using the length-delimited framer
	fw := protobuf.LengthDelimitedFramer.NewFrameWriter(&buf)
	_, err := fw.Write([]byte("frame1"))
	assert.NoError(err)
	_, err = fw.Write([]byte("frame2"))
	assert.NoError(err)

	// Read frames back using the framer reader; ReadAll will return concatenated frames
	fr := protobuf.LengthDelimitedFramer.NewFrameReader(io.NopCloser(bytes.NewReader(buf.Bytes())))
	out, err := io.ReadAll(fr)
	assert.NoError(err)
	assert.Equal("frame1frame2", string(out))
}

func TestJSONStreamEncodeDecode(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}
	streamInfo := factory.SupportedMediaTypes()[0].StreamSerializer
	jsonStreamSerializer := streamInfo.Serializer

	var buf bytes.Buffer
	// Use the JSON framer writer
	fw := streamInfo.Framer.NewFrameWriter(&buf)

	// Write two Pod objects as unstructured
	pod1 := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "p1"}}}
	pod2 := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "p2"}}}

	err := jsonStreamSerializer.Encode(pod1, fw)
	assert.NoError(err)
	err = jsonStreamSerializer.Encode(pod2, fw)
	assert.NoError(err)

	// Split on newlines and decode each non-empty frame. The JSON framer writes
	// newline-delimited JSON objects into the frame writer, so splitting on
	// '\n' recovers individual object bytes.
	frames := bytes.Split(buf.Bytes(), []byte("\n"))
	decodedNames := []string{}
	for _, f := range frames {
		if len(bytes.TrimSpace(f)) == 0 {
			continue
		}
		obj, _, err := jsonStreamSerializer.Decode(f, nil, nil)
		assert.NoError(err)
		u, ok := obj.(*unstructured.Unstructured)
		assert.True(ok)
		decodedNames = append(decodedNames, u.GetName())
	}

	assert.Equal([]string{"p1", "p2"}, decodedNames)
}

func TestProtobufStreamEncodeUnstructuredFailsGracefully(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{
		creator: unstructuredscheme.NewUnstructuredCreator(),
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
	}

	protoStream := factory.SupportedMediaTypes()[2].StreamSerializer
	var buf bytes.Buffer
	fw := protoStream.Framer.NewFrameWriter(&buf)

	pod := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "p1"}}}

	// Encoding unstructured with RawSerializer should fail but not panic.
	assert.NotPanics(func() {
		err := protoStream.Serializer.Encode(pod, fw)
		assert.Error(err)
	})
}

func TestProtobufStreamDecodeArbitraryFrameReturnsError(t *testing.T) {
	assert := assert.New(t)

	factory := WithoutConversionCodecFactory{}
	protoStream := factory.SupportedMediaTypes()[2].StreamSerializer

	// Create a length-delimited framed buffer with arbitrary bytes
	var buf bytes.Buffer
	fw := protoStream.Framer.NewFrameWriter(&buf)
	_, err := fw.Write([]byte{0x01, 0x02, 0x03, 0x04})
	assert.NoError(err)

	// Read the framed payload
	fr := protoStream.Framer.NewFrameReader(io.NopCloser(bytes.NewReader(buf.Bytes())))
	data, err := io.ReadAll(fr)
	assert.NoError(err)

	// Attempt to decode using RawSerializer; should return an error for arbitrary bytes
	_, _, err = protoStream.Serializer.Decode(data, nil, nil)
	assert.Error(err)
}

func TestProtobufStreamEncodeDecodeTypedPod(t *testing.T) {
	assert := assert.New(t)

	// Use a RawSerializer backed by the legacy scheme so typed corev1 objects
	// are supported for protobuf encoding/decoding.
	protoRaw := protobuf.NewRawSerializer(legacyscheme.Scheme, legacyscheme.Scheme)
	framer := protobuf.LengthDelimitedFramer

	// Create a concrete Pod (typed object with protobuf support)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "typed-pod"},
		Spec:       corev1.PodSpec{},
	}

	var buf bytes.Buffer
	fw := framer.NewFrameWriter(&buf)

	// Encode the typed Pod using RawSerializer into a length-delimited frame
	err := protoRaw.Encode(pod, fw)
	assert.NoError(err)

	// Read back the framed payload
	fr := framer.NewFrameReader(io.NopCloser(bytes.NewReader(buf.Bytes())))
	data, err := io.ReadAll(fr)
	assert.NoError(err)

	// Decode using RawSerializer into a typed Pod
	into := &corev1.Pod{}
	obj, _, err := protoRaw.Decode(data, nil, into)
	assert.NoError(err)
	decodedPod, ok := obj.(*corev1.Pod)
	assert.True(ok)
	assert.Equal("typed-pod", decodedPod.Name)
}

func TestProtobufStreamEncodeDecodeTypedPodList(t *testing.T) {
	assert := assert.New(t)

	protoRaw := protobuf.NewRawSerializer(legacyscheme.Scheme, legacyscheme.Scheme)
	framer := protobuf.LengthDelimitedFramer

	// Create a typed PodList with two items
	podList := &corev1.PodList{
		Items: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "pod-a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "pod-b"}},
		},
	}

	var buf bytes.Buffer
	fw := framer.NewFrameWriter(&buf)

	// Encode the PodList
	err := protoRaw.Encode(podList, fw)
	assert.NoError(err)

	// Read back the framed payload
	fr := framer.NewFrameReader(io.NopCloser(bytes.NewReader(buf.Bytes())))
	data, err := io.ReadAll(fr)
	assert.NoError(err)

	// Decode into a typed PodList
	into := &corev1.PodList{}
	obj, _, err := protoRaw.Decode(data, nil, into)
	assert.NoError(err)
	decodedList, ok := obj.(*corev1.PodList)
	assert.True(ok)
	assert.Len(decodedList.Items, 2)
	assert.Equal("pod-a", decodedList.Items[0].Name)
	assert.Equal("pod-b", decodedList.Items[1].Name)
}

// Note: fallbackSerializer and conditionalSerializer types were removed.
// Tests for protobuf behavior now expect protobuf serializer to be used and
// unstructured objects to fail protobuf encoding.

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
