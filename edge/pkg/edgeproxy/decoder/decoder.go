package decoder

import (
	"errors"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	restclientwatch "k8s.io/client-go/rest/watch"
)

// Manager interface provides methods to get the corresponding Decoder based on the resource type.
type Manager interface {
	GetDecoder(contentType string, gv schema.GroupVersion) (runtime.Decoder, error)
	GetStreamDecoder(contentType string, gv schema.GroupVersion, reader io.ReadCloser) (watch.Decoder, error)
}

var DefaultDecoderMgr = &mgr{
	serializer: serializer.NewCodecFactory(scheme.Scheme).WithoutConversion(),
}

type mgr struct {
	serializer runtime.NegotiatedSerializer
}

func (dm *mgr) GetDecoder(contentType string, gv schema.GroupVersion) (runtime.Decoder, error) {
	decoder, _, err := dm.getDecoder(contentType, gv)
	return decoder, err
}

func (dm *mgr) getDecoder(contentType string, gv schema.GroupVersion) (runtime.Decoder, runtime.SerializerInfo, error) {
	mediaTypes := dm.serializer.SupportedMediaTypes()
	info, ok := runtime.SerializerInfoForMediaType(mediaTypes, contentType)
	if !ok {
		if len(contentType) != 0 || len(mediaTypes) == 0 {
			return nil, info, errors.New("content type and midiaTypes'dm length are empty")
		}
		info = mediaTypes[0]
	}
	decoder := dm.serializer.DecoderToVersion(info.Serializer, gv)
	return decoder, info, nil
}

func (dm *mgr) GetStreamDecoder(contentType string, gv schema.GroupVersion, reader io.ReadCloser) (watch.Decoder, error) {
	objDecoder, info, err := dm.getDecoder(contentType, gv)
	if err != nil {
		return nil, err
	}
	frameReader := info.StreamSerializer.Framer.NewFrameReader(reader)
	watchEventDecoder := streaming.NewDecoder(frameReader, info.StreamSerializer)
	watchDecoder := restclientwatch.NewDecoder(watchEventDecoder, objDecoder)
	return watchDecoder, nil
}
