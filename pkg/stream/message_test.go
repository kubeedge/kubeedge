/*
Copyright 2023 The KubeEdge Authors.

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
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"testing"
)

func TestMessageType_String(t *testing.T) {
	tests := []struct {
		name string
		m    MessageType
		want string
	}{
		{
			name: "MessageTypeLogsConnect",
			want: "LOGS_CONNECT",
		},
		{
			name: "MessageTypeExecConnect",
			m:    MessageTypeExecConnect,
			want: "EXEC_CONNECT",
		},
		{
			name: "MessageTypeMetricConnect",
			m:    MessageTypeMetricConnect,
			want: "METRIC_CONNECT",
		},
		{
			name: "MessageTypeData",
			m:    MessageTypeData,
			want: "DATA",
		},
		{
			name: "MessageTypeRemoveConnect",
			m:    MessageTypeRemoveConnect,
			want: "REMOVE_CONNECT",
		},
		{
			name: "Unknown",
			m:    MessageTypeCloseConnect,
			want: "UNKNOWN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.String(); got != tt.want {
				t.Errorf("MessageType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_Bytes(t *testing.T) {
	connID, msgType, data := uint64(1), MessageTypeExecConnect, []byte("test")
	buf, offset := make([]byte, 16), 0
	offset += binary.PutUvarint(buf[offset:], connID)
	offset += binary.PutUvarint(buf[offset:], uint64(msgType))
	buf = append(buf[0:offset], data...)
	tests := []struct {
		name        string
		ConnectID   uint64
		MessageType MessageType
		Data        []byte
		want        []byte
	}{
		{
			name:        "base",
			ConnectID:   connID,
			MessageType: msgType,
			Data:        data,
			want:        buf,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{
				ConnectID:   tt.ConnectID,
				MessageType: tt.MessageType,
				Data:        tt.Data,
			}
			if got := m.Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Message.Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_String(t *testing.T) {
	tests := []struct {
		name        string
		ConnectID   uint64
		MessageType MessageType
		Data        []byte
		want        string
	}{
		{
			name:      "base",
			ConnectID: uint64(1),
			want:      "MESSAGE: connectID 1 messageType LOGS_CONNECT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{
				ConnectID:   tt.ConnectID,
				MessageType: tt.MessageType,
				Data:        tt.Data,
			}
			if got := m.String(); got != tt.want {
				t.Errorf("Message.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadMessageFromTunnel(t *testing.T) {
	connID, msgType, data := uint64(1), MessageTypeExecConnect, []byte("test")
	buf, offset := make([]byte, 16), 0
	offset += binary.PutUvarint(buf[offset:], connID)
	offset += binary.PutUvarint(buf[offset:], uint64(msgType))
	buf = append(buf[0:offset], data...)
	tests := []struct {
		name    string
		r       io.Reader
		want    *Message
		wantErr bool
	}{
		{
			name: "base",
			r:    bytes.NewBuffer(buf),
			want: &Message{
				ConnectID:   connID,
				MessageType: msgType,
				Data:        data,
			},
		},
		{
			name:    "read connID error",
			r:       bytes.NewBuffer([]byte{}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadMessageFromTunnel(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMessageFromTunnel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadMessageFromTunnel() = %v, want %v", got, tt.want)
			}
		})
	}
}
