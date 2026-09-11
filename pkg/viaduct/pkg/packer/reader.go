package packer

import (
	"errors"
	"fmt"
	"io"
	"net"

	"k8s.io/klog/v2"
)

type Reader struct {
	reader io.Reader
}

// silentReadErr reports whether a read error is an expected disconnect path
// that the caller handles and logs itself: EOF (legacy behavior) or a
// deadline expiry (the designed half-open detection in
// conn.WSConnection.handleMessage).
func silentReadErr(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
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
		if !silentReadErr(err) {
			klog.Error("failed to read package header from buffer")
		}
		return nil, err
	}

	header := PackageHeader{}
	header.Unpack(headerBuffer)

	if header.PayloadLen > MaxPayloadLen {
		return nil, fmt.Errorf("payload length %d exceeds maximum %d", header.PayloadLen, MaxPayloadLen)
	}

	payloadBuffer := make([]byte, header.PayloadLen)
	_, err = io.ReadFull(r.reader, payloadBuffer)
	if err != nil {
		if !silentReadErr(err) {
			klog.Error("failed to read payload from buffer")
		}
		return nil, err
	}

	return payloadBuffer, nil
}
