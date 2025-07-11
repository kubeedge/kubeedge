package writer

import (
	"k8s.io/klog/v2"
)

const (
	// WriterTypeRaw writer type raw
	WriterTypeRaw = "raw"
	// WriterTypePackage writer type package
	WriterTypePackage = "package"
)

// Writer writer
type Writer interface {
	Write(message []byte) error
	WriteJSON(obj interface{}) error
}

// NewWriter new writer
func NewWriter(writerType string, conn interface{}) Writer {
	switch writerType {
	case WriterTypeRaw:
		return NewRawWriter(conn)
	case WriterTypePackage:
		return NewPackageWriter(conn)
	}
	klog.Errorf("bad writer type:%s", writerType)
	return nil
}
