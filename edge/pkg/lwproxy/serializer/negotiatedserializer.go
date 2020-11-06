package serializer

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
	"k8s.io/client-go/kubernetes/scheme"
)

var basicScheme = scheme.Scheme

// generalNegotiatedSerializer is used to handle discovery and error handling serialization
type generalNegotiatedSerializer struct{}

func (s generalNegotiatedSerializer) SupportedMediaTypes() []runtime.SerializerInfo {
	return []runtime.SerializerInfo{
		{
			MediaType:        "application/json",
			MediaTypeType:    "application",
			MediaTypeSubType: "json",
			EncodesAsText:    true,
			Serializer:       json.NewSerializerWithOptions(json.DefaultMetaFactory, unstructuredCreater{basicScheme}, unstructuredTyper{basicScheme}, json.SerializerOptions{Yaml: false, Pretty: false, Strict: false}),
			PrettySerializer: json.NewSerializerWithOptions(json.DefaultMetaFactory, unstructuredCreater{basicScheme}, unstructuredTyper{basicScheme}, json.SerializerOptions{Yaml: false, Pretty: true, Strict: false}),
			StreamSerializer: &runtime.StreamSerializerInfo{
				EncodesAsText: true,
				Serializer: json.NewSerializerWithOptions(json.DefaultMetaFactory, unstructuredCreater{basicScheme}, unstructuredTyper{basicScheme}, json.SerializerOptions{Yaml: false, Pretty: false, Strict: false}),
				Framer:     json.Framer,
			},
		},
		{
			MediaType:        "application/yaml",
			MediaTypeType:    "application",
			MediaTypeSubType: "yaml",
			EncodesAsText:    true,
			Serializer:       json.NewSerializerWithOptions(json.DefaultMetaFactory, basicScheme, basicScheme, json.SerializerOptions{Yaml: false, Pretty: false, Strict: false}),
			},
		{
			MediaType:        "application/vnd.kubernetes.protobuf",
			MediaTypeType:    "application",
			MediaTypeSubType: "vnd.kubernetes.protobuf",
			Serializer:       protobuf.NewSerializer(basicScheme, basicScheme),
			StreamSerializer: &runtime.StreamSerializerInfo{
				Serializer: protobuf.NewRawSerializer(basicScheme, basicScheme),
				Framer:     protobuf.LengthDelimitedFramer,
			},
		},
	}
}

func (s generalNegotiatedSerializer) EncoderForVersion(encoder runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder {
	return runtime.WithVersionEncoder{
		Version:     gv,
		Encoder:     encoder,
		ObjectTyper: unstructuredTyper{basicScheme},
	}
}

func (s generalNegotiatedSerializer) DecoderToVersion(decoder runtime.Decoder, gv runtime.GroupVersioner) runtime.Decoder {
	return decoder
}

type unstructuredCreater struct {
	nested runtime.ObjectCreater
}

func (c unstructuredCreater) New(kind schema.GroupVersionKind) (runtime.Object, error) {
	out, err := c.nested.New(kind)
	if err == nil {
		return out, nil
	}
	out = &unstructured.Unstructured{}
	out.GetObjectKind().SetGroupVersionKind(kind)
	return out, nil
}

type unstructuredTyper struct {
	nested runtime.ObjectTyper
}

func (t unstructuredTyper) ObjectKinds(obj runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	kinds, unversioned, err := t.nested.ObjectKinds(obj)
	if err == nil {
		return kinds, unversioned, nil
	}
	if _, ok := obj.(runtime.Unstructured); ok && !obj.GetObjectKind().GroupVersionKind().Empty() {
		return []schema.GroupVersionKind{obj.GetObjectKind().GroupVersionKind()}, false, nil
	}
	return nil, false, err
}

func (t unstructuredTyper) Recognizes(gvk schema.GroupVersionKind) bool {
	return true
}
