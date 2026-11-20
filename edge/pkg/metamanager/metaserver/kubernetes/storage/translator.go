package storage

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

func DecodeAndConvert(data []byte) (runtime.Object, error) {
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)
	decoder := codecFactory.UniversalDeserializer()

	unstructuredObj := &unstructured.Unstructured{}

	obj, _, err := decoder.Decode(data, nil, unstructuredObj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
