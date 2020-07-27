package util

import (
	"io"

	"go.uber.org/atomic"
	"k8s.io/klog"
)

type ReadCloseDuplicater interface {
	io.ReadCloser
	DupData() io.ReadCloser
}

func NewDuplicateReadCloser(source io.ReadCloser) ReadCloseDuplicater {
	return &duplicateReader{
		source:   source,
		dataChan: make(chan []byte),
	}
}

type duplicateReader struct {
	source   io.ReadCloser
	dataChan chan []byte
}

func (d *duplicateReader) Close() error {
	close(d.dataChan)
	return d.source.Close()
}

func (d *duplicateReader) Read(p []byte) (n int, err error) {
	n, err = d.source.Read(p)
	if n > 0 {
		d.dataChan <- p[:n]
	}
	return
}

func (d *duplicateReader) DupData() io.ReadCloser {
	return newBytesChanReadCloser(d.dataChan)
}

func newBytesChanReadCloser(byteChan chan []byte) io.ReadCloser {
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
			klog.Info("data chan is closed, return io.EOF")
			return 0, io.EOF
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
