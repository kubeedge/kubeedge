package packer

import (
	"fmt"
	"io"

	"k8s.io/klog/v2"
)

type Writer struct {
	writer io.Writer
}

// new Writer instance
func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

// Write message raw data
// steps:
// 1) packer the package header
// 2) write header
// 3) write message raw data
func (w *Writer) Write(data []byte) (int, error) {
	if w.writer == nil {
		klog.Error("bad io writer")
		return 0, fmt.Errorf("bad io writer")
	}
	if len(data) > int(MaxPayloadLen) {
		return 0, fmt.Errorf("payload length %d exceeds maximum %d", len(data), MaxPayloadLen)
	}

	// packing header
	header := NewPackageHeader(Message)
	header.SetPayloadLen(uint32(len(data)))
	var headerBuffer []byte
	header.Pack(&headerBuffer)

	// write header
	_, err := w.writer.Write(headerBuffer)
	if err != nil {
		klog.Error("failed to write header")
		return 0, err
	}

	// write payload
	_, err = w.writer.Write(data)
	if err != nil {
		klog.Error("failed to write payload")
		return 0, err
	}
	return len(data), nil
}
