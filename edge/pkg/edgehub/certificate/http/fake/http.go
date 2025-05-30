package fake

import "bytes"

type FakeBodyReader struct {
	*bytes.Reader
}

func NewFakeBodyReader(bff []byte) *FakeBodyReader {
	return &FakeBodyReader{
		Reader: bytes.NewReader(bff),
	}
}

func (FakeBodyReader) Close() error {
	return nil
}
