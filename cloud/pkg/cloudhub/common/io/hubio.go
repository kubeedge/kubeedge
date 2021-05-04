package io

import (
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/conn"
)

// CloudHubIO handle the IO operation from connection
type CloudHubIO interface {
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	ReadData(*model.Message) (int, error)
	WriteData(*model.Message) error
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
func (io *JSONIO) ReadData(msg *model.Message) (int, error) {
	return 0, io.Connection.ReadMessage(msg)
}

// WriteData write data to connection
func (io *JSONIO) WriteData(msg *model.Message) error {
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
