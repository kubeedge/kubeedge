package packer

import (
	"fmt"
	"io"

	"github.com/kubeedge/beehive/pkg/common/log"
)

type Reader struct {
	reader io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{reader: r}
}

func (r *Reader) Read() ([]byte, error) {
	if r.reader == nil {
		log.LOGGER.Errorf("bad io reader")
		return nil, fmt.Errorf("bad io reader")
	}

	headerBuffer := make([]byte, HeaderSize)
	_, err := io.ReadFull(r.reader, headerBuffer)
	if err != nil {
		if err != io.EOF {
			log.LOGGER.Errorf("failed to read package header from buffer")
		}
		return nil, err
	}

	header := PackageHeader{}
	header.Unpack(headerBuffer)

	payloadBuffer := make([]byte, header.PayloadLen)
	_, err = io.ReadFull(r.reader, payloadBuffer)
	if err != nil {
		if err != io.EOF {
			log.LOGGER.Errorf("failed to read payload from buffer")
		}
		return nil, err
	}

	return payloadBuffer, nil
}
