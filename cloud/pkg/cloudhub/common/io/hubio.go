package io

import (
	"fmt"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/conn"
)

// CloudHubIO handle the IO operation from connection
type CloudHubIO interface {
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	ReadData(interface{}) (int, error)
	WriteData(interface{}) error
	Close() error
}

// JSONIO address the json data from connection
type JSONIO struct {
	Connection conn.Connection
}

// SetReadDeadline set read operation dead line
func (io *JSONIO) SetReadDeadline(time time.Time) error {
	return io.Connection.SetReadDeadline(time)
}

// SetWriteDeadline set write operation dead line
func (io *JSONIO) SetWriteDeadline(time time.Time) error {
	return io.Connection.SetWriteDeadline(time)
}

// ReadData read data from connection
func (io *JSONIO) ReadData(d interface{}) (int, error) {
	return 0, io.Connection.ReadMessage(d.(*model.Message))
}

// WriteData write data to connection
func (io *JSONIO) WriteData(d interface{}) error {
	msg, ok := d.(*model.Message)
	if !ok {
		return fmt.Errorf("data is not model.Message type")
	}
	err := io.Connection.WriteMessageAsync(msg)
	if err != nil {
		return err
	}
	return nil
}

// Close close the IO operation
func (io *JSONIO) Close() error {
	return io.Connection.Close()
}
