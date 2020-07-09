package decoder

import (
	"errors"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/client-go/kubernetes/scheme"
	restclientwatch "k8s.io/client-go/rest/watch"
)

type DecoderMgr interface {
	GetDecoder(contentType string, gv schema.GroupVersion) (runtime.Decoder, error)
	GetStreamDecocer(contentType string, gv schema.GroupVersion, reader io.ReadCloser) (*restclientwatch.Decoder, error)
}

var DefaultDecoderMgr = &decoderMgr{
	serializer: serializer.NewCodecFactory(scheme.Scheme).WithoutConversion(),
}

type decoderMgr struct {
	serializer runtime.NegotiatedSerializer
}

func (dm *decoderMgr) GetDecoder(contentType string, gv schema.GroupVersion) (runtime.Decoder, error) {
	decoder, _, err := dm.decoder(contentType, gv)
	return decoder, err
}

func (dm *decoderMgr) decoder(contentType string, gv schema.GroupVersion) (runtime.Decoder, runtime.SerializerInfo, error) {
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

func (dm *decoderMgr) GetStreamDecocer(contentType string, gv schema.GroupVersion, reader io.ReadCloser) (*restclientwatch.Decoder, error) {
	objDecoder, info, err := dm.decoder(contentType, gv)
	if err != nil {
		return nil, err
	}
	framereader := info.StreamSerializer.Framer.NewFrameReader(reader)
	watchEventDecoder := streaming.NewDecoder(framereader, info.StreamSerializer)
	watchDecoder := restclientwatch.NewDecoder(watchEventDecoder, objDecoder)
	return watchDecoder, nil
}
