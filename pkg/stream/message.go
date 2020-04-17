package stream

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gorilla/websocket"
	"k8s.io/klog"
)

type MessageType uint64

const (
	MessageTypeLogsConnect MessageType = iota
	MessageTypeExecConnect MessageType = iota
	MessageTypeData
	MessageTypeRemoveConnect
)

func (m MessageType) String() string {
	switch m {
	case MessageTypeLogsConnect:
		return "LOGS_CONNECT"
	case MessageTypeExecConnect:
		return "EXEC_CONNECT"
	case MessageTypeData:
		return "DATA"
	case MessageTypeRemoveConnect:
		return "REMOVE_CLIENT"
	}
	return "UNKNOWN"
}

type Message struct {
	ConnectID   uint64 // apiserver connection id
	MessageType MessageType
	Data        []byte // EdgeLogsConnector 或者 con 的原始数据
}

func NewMessage(id uint64, messType MessageType, data []byte) *Message {
	return &Message{
		ConnectID:   id,
		MessageType: messType,
		Data:        data,
	}
}

func (m *Message) WriteTo(con *websocket.Conn) error {
	return con.WriteMessage(websocket.TextMessage, m.Bytes())
}

func (m *Message) Bytes() []byte {
	// connectID + MessageType + Data
	buf, offset := make([]byte, 16), 0
	offset += binary.PutUvarint(buf[offset:], m.ConnectID)
	offset += binary.PutUvarint(buf[offset:], uint64(m.MessageType))
	return append(buf[0:offset], m.Data...)
}

func (m *Message) String() string {
	return fmt.Sprintf("MESSAGE: connectid %v MessageType %v", m.ConnectID, m.MessageType)
}

func TunnelMessage(r io.Reader) (*Message, error) {
	buf := bufio.NewReader(r)
	connectID, err := binary.ReadUvarint(buf)
	if err != nil {
		return nil, err
	}
	messageType, err := binary.ReadUvarint(buf)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(buf)
	if err != nil {
		return nil, err
	}
	klog.Infof("Receive Tunnel message Connectid %d messageType %v data %v string:[ %v ]",
		connectID, messageType, data, string(data))
	return &Message{
		ConnectID:   connectID,
		MessageType: MessageType(messageType),
		Data:        data,
	}, nil
}
