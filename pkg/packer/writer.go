package packer

import (
	"fmt"
	"io"

	"github.com/kubeedge/beehive/pkg/common/log"
)

type Writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

func (w *Writer) Write(data []byte) error {
	if w.writer == nil {
		log.LOGGER.Errorf("bad io writer")
		return fmt.Errorf("bad io writer")
	}

	// packing header
	header := NewPackageHeader(Message)
	header.SetPayloadLen(uint32(len(data)))
	var headerBuffer []byte
	header.Pack(&headerBuffer)

	// write header
	_, err := w.writer.Write(headerBuffer)
	if err != nil {
		log.LOGGER.Errorf("failed to write header")
		return err
	}

	// write payload
	_, err = w.writer.Write(data)
	if err != nil {
		log.LOGGER.Errorf("failed to write payload")
		return err
	}
	return nil
}
