package io

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

// CloudHubIO handle the IO operation from connection
type CloudHubIO interface {
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	ReadData(interface{}) (int, error)
	WriteData(interface{}) error
	Close() error
}

// JSONWSIO address the json data from connection
type JSONWSIO struct {
	WSConn *websocket.Conn
}

// SetReadDeadline set read operation dead line
func (io *JSONWSIO) SetReadDeadline(time time.Time) error {
	return io.WSConn.SetReadDeadline(time)
}

// SetWriteDeadline set write operation dead line
func (io *JSONWSIO) SetWriteDeadline(time time.Time) error {
	return io.WSConn.SetWriteDeadline(time)
}

// ReadData read data from connection
func (io *JSONWSIO) ReadData(d interface{}) (int, error) {
	_, buf, err := io.WSConn.ReadMessage()
	if err != nil {
		return 0, err
	}
	err = json.Unmarshal(buf, d)
	return len(buf), err
}

// WriteData write data to connection
func (io *JSONWSIO) WriteData(d interface{}) error {
	return io.WSConn.WriteJSON(d)
}

// Close close the IO operation
func (io *JSONWSIO) Close() error {
	return io.WSConn.Close()
}
