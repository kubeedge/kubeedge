package io

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
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
	var data []byte
	var err error

	switch d.(type) {
	case []byte:
		data = d.([]byte)
	default:
		data, err = json.Marshal(d)
		if err != nil {
			return fmt.Errorf("websocket write msg failed with marshal failed. error %s", err.Error())
		}
	}
	return io.WSConn.WriteMessage(websocket.BinaryMessage, data)
}

// Close close the IO operation
func (io *JSONWSIO) Close() error {
	return io.WSConn.Close()
}

type JSONQuicIO struct {
	Connection conn.Connection
}

func (quicio *JSONQuicIO) SetReadDeadline(time time.Time) error {
	return nil
}

func (quicio *JSONQuicIO) SetWriteDeadline(time time.Time) error {
	return nil
}

func (quicio *JSONQuicIO) ReadData(d interface{}) (int, error) {
	return 0, quicio.Connection.ReadMessage(d.(*model.Message))
}

func (quicio *JSONQuicIO) WriteData(d interface{}) error {
	msg, ok := d.(*model.Message)
	if !ok {
		return fmt.Errorf("data is not model.Message type")
	}
	err := quicio.Connection.WriteMessageAsync(msg)
	if err != nil {
		return err
	}
	return nil
}

func (quicio *JSONQuicIO) Close() error {
	return quicio.Connection.Close()
}
