package wrapper

import (
	"net"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/socket/wrapper/reader"
	"github.com/kubeedge/beehive/pkg/core/socket/wrapper/writer"
)

// Conn conn
type Conn interface {
	Read() ([]byte, error)
	Write(message []byte) error

	ReadJSON(obj interface{}) error
	WriteJSON(obj interface{}) error

	Close() error

	SetReadDeadline(t time.Time) error
}

// ConnWrapper conn wrapper
type ConnWrapper struct {
	conn   interface{}
	reader reader.Reader
	writer writer.Writer
}

// NewWrapper new wrapper
func NewWrapper(connType string, conn interface{}, buffSize int) Conn {
	readerType := reader.ReaderTypeRaw
	writerType := writer.WriterTypeRaw

	return &ConnWrapper{
		conn:   conn,
		reader: reader.NewReader(readerType, conn, buffSize),
		writer: writer.NewWriter(writerType, conn),
	}
}

// Read read
func (w *ConnWrapper) Read() ([]byte, error) {
	return w.reader.Read()
}

// Write write
func (w *ConnWrapper) Write(message []byte) error {
	return w.writer.Write(message)
}

// ReadJSON read json
func (w *ConnWrapper) ReadJSON(obj interface{}) error {
	return w.reader.ReadJSON(obj)
}

// WriteJSON write json
func (w *ConnWrapper) WriteJSON(obj interface{}) error {
	return w.writer.WriteJSON(obj)
}

// SetReadDeadline set read deadline
func (w *ConnWrapper) SetReadDeadline(t time.Time) error {
	// TODO: put int Deadline
	var err error
	switch w.conn.(type) {
	case net.Conn:
		conn := w.conn.(net.Conn)
		err = conn.SetReadDeadline(t)
	default:
		klog.Warning("unsupported conn type: %T", w.conn)
	}
	return err
}

// Close close
func (w *ConnWrapper) Close() error {
	// TODO: put into Closer
	var err error
	switch w.conn.(type) {
	case net.Conn:
		conn := w.conn.(net.Conn)
		err = conn.Close()
	default:
		klog.Warning("unsupported conn type: %T", w.conn)
	}
	return err
}
