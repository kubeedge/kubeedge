package util

import (
	"fmt"
	"io"

	"go.uber.org/atomic"
)

func NewBytesChanReadCloser(byteChan chan []byte) io.ReadCloser {
	return &byteReadCloser{dataChan: byteChan, bufClear: atomic.NewBool(true)}
}

type byteReadCloser struct {
	dataChan chan []byte
	buf      []byte
	bufClear *atomic.Bool
}

func (b *byteReadCloser) Read(p []byte) (n int, err error) {
	if b.bufClear.Load() {
		buf, ok := <-b.dataChan
		if !ok {
			return 0, fmt.Errorf("read error")
		}
		b.buf = buf
		b.bufClear.Store(false)
	}

	n = copy(p, b.buf)
	b.buf = b.buf[n:]
	if len(b.buf) == 0 {
		b.bufClear.Store(true)
	}
	return n, nil
}

func (b *byteReadCloser) Close() error {
	close(b.dataChan)
	return nil
}
