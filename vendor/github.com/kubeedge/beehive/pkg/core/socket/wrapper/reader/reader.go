package reader

import (
	"k8s.io/klog/v2"
)

const (
	// ReaderTypeRaw reader type raw
	ReaderTypeRaw = "raw"
	// ReaderTypePackage reader type package
	ReaderTypePackage = "package"
)

// Reader reader
type Reader interface {
	Read() ([]byte, error)
	ReadJSON(obj interface{}) error
}

// NewReader new reader
func NewReader(readerType string, conn interface{}, buffSize int) Reader {
	switch readerType {
	case ReaderTypeRaw:
		return NewRawReader(conn, buffSize)
	case ReaderTypePackage:
		return NewPackageReader(conn, buffSize)
	}
	klog.Errorf("bad reader type: %s", readerType)
	return nil
}
