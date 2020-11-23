/*
Copyright 2020 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package stream

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"k8s.io/klog/v2"
)

type MessageType uint64

const (
	MessageTypeLogsConnect MessageType = iota
	MessageTypeExecConnect
	MessageTypeMetricConnect
	MessageTypeData
	MessageTypeRemoveConnect
)

func (m MessageType) String() string {
	switch m {
	case MessageTypeLogsConnect:
		return "LOGS_CONNECT"
	case MessageTypeExecConnect:
		return "EXEC_CONNECT"
	case MessageTypeMetricConnect:
		return "METRIC_CONNECT"
	case MessageTypeData:
		return "DATA"
	case MessageTypeRemoveConnect:
		return "REMOVE_CONNECT"
	}
	return "UNKNOWN"
}

type Message struct {
	// ConnectID indicate the apiserver connection id
	ConnectID   uint64
	MessageType MessageType
	Data        []byte
}

func NewMessage(id uint64, messType MessageType, data []byte) *Message {
	return &Message{
		ConnectID:   id,
		MessageType: messType,
		Data:        data,
	}
}

func (m *Message) WriteTo(tunneler SafeWriteTunneler) error {
	return tunneler.WriteMessage(m)
}

func (m *Message) Bytes() []byte {
	// connectID + MessageType + Data
	buf, offset := make([]byte, 16), 0
	offset += binary.PutUvarint(buf[offset:], m.ConnectID)
	offset += binary.PutUvarint(buf[offset:], uint64(m.MessageType))
	return append(buf[0:offset], m.Data...)
}

func (m *Message) String() string {
	return fmt.Sprintf("MESSAGE: connectid %v MessageType %s", m.ConnectID, m.MessageType)
}

func ReadMessageFromTunnel(r io.Reader) (*Message, error) {
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
	klog.V(6).Infof("Receive Tunnel message Connectid %d messageType %s data:%v string:[%v]",
		connectID, MessageType(messageType), data, string(data))
	return &Message{
		ConnectID:   connectID,
		MessageType: MessageType(messageType),
		Data:        data,
	}, nil
}
