package writer

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"k8s.io/klog/v2"
)

// RawWriter raw writer
type RawWriter struct {
	conn net.Conn
	lock sync.Mutex
}

// NewRawWriter new raw writer
func NewRawWriter(obj interface{}) *RawWriter {
	if conn, ok := obj.(net.Conn); ok {
		return &RawWriter{conn: conn}
	}
	klog.Errorf("bad conn ")
	return nil
}

// Write write
func (w *RawWriter) Write(message []byte) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	number, err := w.conn.Write(message)
	if err != nil || number != len(message) {
		klog.Errorf("failed to write, error:%+v", err)
		return fmt.Errorf("failed to write, error: %+v", err)
	}
	return nil
}

// WriteJSON write json
func (w *RawWriter) WriteJSON(obj interface{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	return json.NewEncoder(w.conn).Encode(obj)
}
