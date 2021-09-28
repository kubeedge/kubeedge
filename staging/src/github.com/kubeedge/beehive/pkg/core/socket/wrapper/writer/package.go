package writer

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/socket/wrapper/packer"
)

// PackageWriter package writer
type PackageWriter struct {
	packer *packer.Packer
	conn   net.Conn
	lock   sync.Mutex
}

// NewPackageWriter new package writer
func NewPackageWriter(obj interface{}) *PackageWriter {
	if conn, ok := obj.(net.Conn); ok {
		packer := packer.NewPacker()
		return &PackageWriter{
			conn:   conn,
			packer: packer,
		}
	}
	klog.Errorf("bad conn obj")
	return nil
}

// Write write
func (w *PackageWriter) Write(message []byte) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.packer.Message = message
	w.packer.Length = int32(len(message))
	err := w.packer.Write(w.conn)
	if err != nil {
		klog.Errorf("failed to packer with error %+v", err)
		return fmt.Errorf("failed to packer, error:%+v", err)
	}
	return nil
}

// WriteJSON write json
func (w *PackageWriter) WriteJSON(obj interface{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	objBytes, err := json.Marshal(obj)
	if err != nil {
		klog.Errorf("failed to marshal obj, error:%+v", err)
		return err
	}
	w.packer.Message = objBytes
	w.packer.Length = int32(len(objBytes))
	err = w.packer.Write(w.conn)
	if err != nil {
		klog.Errorf("failed to packer, error:%+v", err)
		return fmt.Errorf("failed to packer, error:%+v", err)
	}
	return nil
}
