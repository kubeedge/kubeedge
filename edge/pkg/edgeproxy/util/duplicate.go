package util

import (
	"io"
)

type ReadCloseDuplicater interface {
	io.ReadCloser
	DupData() chan []byte
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

// readonly chan but can be closed. need refactor
func (d *duplicateReader) DupData() chan []byte {
	return d.dataChan
}
