package serializer

import (
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// NewUnstructuredNegotiatedSerializer returns a simple, negotiated serializer
func NewNegotiatedSerializer() runtime.NegotiatedSerializer {
	return WithoutConversionCodecFactory{
		typer:   unstructuredscheme.NewUnstructuredObjectTyper(),
		creator: unstructuredscheme.NewUnstructuredCreator(),
	}
}

type WithoutConversionCodecFactory struct {
	creator runtime.ObjectCreater
	typer   runtime.ObjectTyper
}

func (f WithoutConversionCodecFactory) SupportedMediaTypes() []runtime.SerializerInfo {
	return []runtime.SerializerInfo{
		{
			MediaType:        "application/json",
			MediaTypeType:    "application",
			MediaTypeSubType: "json",
			EncodesAsText:    true,
			Serializer:       json.NewSerializerWithOptions(json.DefaultMetaFactory, f.creator, f.typer, json.SerializerOptions{Pretty: false}),
			PrettySerializer: json.NewSerializerWithOptions(json.DefaultMetaFactory, f.creator, f.typer, json.SerializerOptions{Pretty: true}),
			StreamSerializer: &runtime.StreamSerializerInfo{
				EncodesAsText: true,
				Serializer:    json.NewSerializerWithOptions(json.DefaultMetaFactory, f.creator, f.typer, json.SerializerOptions{Pretty: false}),
				Framer:        json.Framer,
			},
		},
		{
			MediaType:        "application/yaml",
			MediaTypeType:    "application",
			MediaTypeSubType: "yaml",
			EncodesAsText:    true,
			Serializer:       json.NewYAMLSerializer(json.DefaultMetaFactory, f.creator, f.typer),
		},
	}
}

// EncoderForVersion return an encoder that set the obj's GVK before encode
func (f WithoutConversionCodecFactory) EncoderForVersion(serializer runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder {
	encoder := &SetVersionEncoder{
		Version: gv,
		encoder: serializer,
	}
	return encoder
}

// DecoderToVersion do nothing but return decoder
func (f WithoutConversionCodecFactory) DecoderToVersion(serializer runtime.Decoder, _ runtime.GroupVersioner) runtime.Decoder {
	return serializer
}

// SetVersionEncoder set the obj's gvk before encode if embed a WithKindGroupVersioner
type SetVersionEncoder struct {
	Version runtime.GroupVersioner
	encoder runtime.Encoder
}

func (s *SetVersionEncoder) Identifier() runtime.Identifier {
	return runtime.Identifier("SetVersionEncoder:" + s.Version.Identifier())
}

func (s *SetVersionEncoder) Encode(obj runtime.Object, w io.Writer) error {
	if gv, ok := s.Version.(*WithKindGroupVersioner); ok {
		gvk, _ := gv.KindForGroupVersionKinds(nil)
		obj.GetObjectKind().SetGroupVersionKind(gvk)
	}
	return s.encoder.Encode(obj, w)
}

type AdapterDecoder struct {
	runtime.Decoder
}

// sometimes
func (d *AdapterDecoder) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	out, gvk, err := d.Decoder.Decode(data, defaults, into)
	if err != nil && gvk != nil {
		*defaults = *gvk
	}
	return out, gvk, err
}

// WithKindGroupVersioner always return embedded gvk when we call KindForGroupVersionKinds
type WithKindGroupVersioner struct {
	gvk schema.GroupVersionKind
}

func (s *WithKindGroupVersioner) KindForGroupVersionKinds(kinds []schema.GroupVersionKind) (target schema.GroupVersionKind, ok bool) {
	return s.gvk, true
}
func (s *WithKindGroupVersioner) Identifier() string {
	return s.gvk.String()
}

func NewWithKindGroupVersioner(gvk schema.GroupVersionKind) *WithKindGroupVersioner {
	gv := &WithKindGroupVersioner{
		gvk: gvk,
	}
	return gv
}
