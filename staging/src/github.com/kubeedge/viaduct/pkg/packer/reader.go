package packer

import (
	"fmt"
	"io"

	"k8s.io/klog/v2"
)

const (
	maxPayloadLen = 32 * 1 << 20 // 32Mi
)

type Reader struct {
	reader io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{reader: r}
}

// Read message raw data from reader
// steps:
// 1)read the package header
// 2)unpack the package header and get the payload length
// 3)read the payload
func (r *Reader) Read() ([]byte, error) {
	if r.reader == nil {
		klog.Error("bad io reader")
		return nil, fmt.Errorf("bad io reader")
	}

	headerBuffer := make([]byte, HeaderSize)
	_, err := io.ReadFull(r.reader, headerBuffer)
	if err != nil {
		if err != io.EOF {
			klog.Errorf("failed to read package header from buffer: %v", err)
		}
		return nil, err
	}

	header := PackageHeader{}
	header.Unpack(headerBuffer)

	// Aviod huge payloads from causing OOM
	if header.PayloadLen > maxPayloadLen {
		err := fmt.Errorf("failed to read package header: invalid header PayloadLen, max value %d, but PayloadLen got %d", maxPayloadLen, header.PayloadLen)
		klog.Error(err)
		return nil, err
	}

	payloadBuffer := make([]byte, header.PayloadLen)
	_, err = io.ReadFull(r.reader, payloadBuffer)
	if err != nil {
		if err != io.EOF {
			klog.Errorf("failed to read payload from buffer: %v", err)
		}
		return nil, err
	}

	return payloadBuffer, nil
}
