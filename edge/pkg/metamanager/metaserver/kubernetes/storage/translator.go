package storage

import (
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

// decodeAndConvert uses the appropriate scheme to decode the raw JSON data into an
// internal object and then converts it to its external (versioned) representation.
// This is necessary because the protobuf serializer requires a typed, external object.
func DecodeAndConvert(body []byte, group string) (runtime.Object, error) {
	var decoder runtime.Decoder

	switch group {
	case "apiextensions.k8s.io":
		decoder = apiextensionsscheme.Codecs.UniversalDecoder(apiextensionsv1.SchemeGroupVersion)
	case "":
		decoder = legacyscheme.Codecs.UniversalDeserializer()
	default:
		decoder = kubescheme.Codecs.UniversalDecoder(kubescheme.Scheme.PrioritizedVersionsAllGroups()...)
	}
	internalObj, gvk, err := decoder.Decode(body, nil, nil)
	if err != nil {
		// If the type is not registered in the scheme (e.g. a CRD), we can't convert it to a typed object
		// for protobuf serialization. In this case, we fall back to returning a runtime.Unknown object,
		// which will be serialized as JSON.
		if runtime.IsNotRegisteredError(err) {
			//if we are seeing this for a core api resource, then there is a problem with decoding
			if strings.Contains(gvk.Group, ".k8s.io") || gvk.Group == "" {
				klog.V(4).Infof("failed to decode core k8s object, this should not happen. Falling back to runtime.Unknown for gvk: %v, err: %v", gvk, err)
				return &runtime.Unknown{Raw: body, ContentType: runtime.ContentTypeJSON}, nil
			}
			//crd's wont be registered so pass thru
			return &runtime.Unknown{Raw: body, ContentType: runtime.ContentTypeJSON}, nil
		}

		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return internalObj, nil
}
