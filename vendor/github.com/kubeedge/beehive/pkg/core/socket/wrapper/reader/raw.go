package reader

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"k8s.io/klog/v2"
)

// RawReader raw reader
type RawReader struct {
	conn     net.Conn
	lock     sync.Mutex
	buffer   []byte
	buffSize int
}

// NewRawReader new raw reader
func NewRawReader(conn interface{}, buffSize int) *RawReader {
	if conn, ok := conn.(net.Conn); ok {
		return &RawReader{
			conn:     conn,
			buffSize: buffSize,
		}
	}
	klog.Warning("bad conn interface")
	return nil
}

// Read read
func (r *RawReader) Read() ([]byte, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.buffer == nil {
		r.buffer = make([]byte, r.buffSize)
	}

	nr, err := r.conn.Read(r.buffer)
	if err != nil {
		klog.Errorf("failed to read, error %+v", err)
		return nil, fmt.Errorf("failed to read, error: %+v", err)
	}
	return r.buffer[:nr], nil
}

// ReadJSON read json
func (r *RawReader) ReadJSON(obj interface{}) error {
	//r.lock.Lock()
	//defer r.lock.Unlock()
	//return json.NewDecoder(r.conn).Decode(obj)
	buf, err := r.Read()
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, obj)
	if err != nil {
		klog.Errorf("failed to unmarshal message, context: %s, errpr: %+v", string(buf), err)
		return fmt.Errorf("failed to unmarshal message, error:%+v, context: %+v", err, string(buf))
	}
	return nil
}
