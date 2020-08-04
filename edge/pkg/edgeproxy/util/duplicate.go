package util

import (
	"io"

	"go.uber.org/atomic"
	"k8s.io/klog"
)

// Duplicate ReadCloser interface
type ReadCloseDuplicater interface {
	io.ReadCloser
	DupReadCloser() io.ReadCloser
}

func NewDuplicateReadCloser(source io.ReadCloser) ReadCloseDuplicater {
	return &duplicateReader{
		source:   source,
		dataChan: make(chan []byte),
	}
}

type duplicateReader struct {
	source io.ReadCloser
	// copy source data to dataChan
	dataChan chan []byte
}

// close dataChan and source
func (d *duplicateReader) Close() error {
	close(d.dataChan)
	return d.source.Close()
}

// Read data from source and write data to dataChan
func (d *duplicateReader) Read(p []byte) (n int, err error) {
	n, err = d.source.Read(p)
	if n > 0 {
		d.dataChan <- p[:n]
	}
	return
}

// wrap dataChan into io.ReadCloser interface
func (d *duplicateReader) DupReadCloser() io.ReadCloser {
	return newBytesChanReadCloser(d.dataChan)
}

func newBytesChanReadCloser(byteChan chan []byte) io.ReadCloser {
	return &byteReadCloser{dataChan: byteChan, bufClear: atomic.NewBool(true)}
}

// wrap byte chan into io.ReadCloser interface
type byteReadCloser struct {
	dataChan chan []byte
	// store the data received from dataChan.
	buf []byte
	// indicates the length of buf is 0
	bufClear *atomic.Bool
}

func (b *byteReadCloser) Read(p []byte) (n int, err error) {
	if b.bufClear.Load() {
		buf, ok := <-b.dataChan
		if !ok {
			// The closed dataChan indicates that the source has beed Closed. so it needs return io.EOF error
			klog.Info("dataChan has beed closed! return io.EOF")
			return 0, io.EOF
		}
		b.buf = buf
		b.bufClear.Store(false)
	}
	// The length of buf and p may be different, and there may be data that has not been copied in buf
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
