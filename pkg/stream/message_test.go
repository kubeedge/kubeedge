/*
Copyright 2024 The KubeEdge Authors.

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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestMessageType_String(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		msg       MessageType
		stdResult string
	}{
		{
			msg:       MessageTypeLogsConnect,
			stdResult: "LOGS_CONNECT",
		},
		{
			msg:       MessageTypeExecConnect,
			stdResult: "EXEC_CONNECT",
		},
		{
			msg:       MessageTypeAttachConnect,
			stdResult: "ATTACH_CONNECT",
		},
		{
			msg:       MessageTypeMetricConnect,
			stdResult: "METRIC_CONNECT",
		},
		{
			msg:       MessageTypeData,
			stdResult: "DATA",
		},
		{
			msg:       MessageTypeRemoveConnect,
			stdResult: "REMOVE_CONNECT",
		},
		{
			msg:       100,
			stdResult: "UNKNOWN",
		},
	}

	for _, test := range cases {
		assert.Equal(test.stdResult, test.msg.String())
	}
}

func TestMessage_NewMessage(t *testing.T) {
	assert := assert.New(t)
	msg := NewMessage(100, MessageTypeLogsConnect, []byte("test data"))
	assert.Equal(msg.ConnectID, uint64(100))
	assert.Equal(msg.MessageType, MessageTypeLogsConnect)
	assert.Equal(msg.Data, []byte("test data"))
}

func TestMessage_Bytes(t *testing.T) {
	assert := assert.New(t)
	message := NewMessage(1, MessageTypeLogsConnect, []byte("test_data"))

	var stdResult []byte
	buf := make([]byte, 10)
	n := binary.PutUvarint(buf, message.ConnectID)
	stdResult = append(stdResult, buf[:n]...)
	n = binary.PutUvarint(buf, uint64(message.MessageType))
	stdResult = append(stdResult, buf[:n]...)
	stdResult = append(stdResult, message.Data...)

	assert.Equal(stdResult, message.Bytes())
}

func TestMessage_String(t *testing.T) {
	assert := assert.New(t)
	msg := &Message{
		ConnectID:   100,
		MessageType: MessageTypeLogsConnect,
		Data:        []byte("test data"),
	}

	result := msg.String()
	stdResult := "MESSAGE: connectID 100 messageType LOGS_CONNECT"
	assert.Equal(result, stdResult)
}

func TestReadMessageFromTunnel(t *testing.T) {
	assert := assert.New(t)

	message := NewMessage(100, MessageTypeLogsConnect, []byte("test data"))
	messageBytes := message.Bytes()

	buffer := bytes.NewBuffer(messageBytes)

	readMessage, err := ReadMessageFromTunnel(buffer)
	assert.NoError(err)
	assert.Equal(readMessage.ConnectID, uint64(100))
	assert.Equal(readMessage.MessageType, MessageTypeLogsConnect)
	assert.Equal(readMessage.Data, []byte("test data"))
}

func TestReadMessageFromTunnel_ConnectIDError(t *testing.T) {
	assert := assert.New(t)
	errorReader := &errorReader{err: io.ErrClosedPipe}
	_, err := ReadMessageFromTunnel(errorReader)
	assert.ErrorContains(err, "closed pipe")
}

func TestReadMessageFromTunnel_MessageTypeError(t *testing.T) {
	assert := assert.New(t)
	buf := bytes.NewBuffer([]byte{0x01}) // Valid connectID
	buf.WriteByte(0x80)                  // Start multi-byte varint
	errorReader := io.MultiReader(buf, &errorReader{err: io.ErrUnexpectedEOF})

	_, err := ReadMessageFromTunnel(errorReader)
	assert.ErrorContains(err, "unexpected EOF")
}

func TestReadMessageFromTunnel_DataReadError(t *testing.T) {
	assert := assert.New(t)

	headerBuf := bytes.NewBuffer(nil)
	err := binary.Write(headerBuf, binary.LittleEndian, uint64(1)) // ConnectID
	assert.NoError(err, "should write connect ID")
	err = binary.Write(headerBuf, binary.LittleEndian, uint64(2)) // MessageType
	assert.NoError(err, "should write message type")

	errorReader := io.MultiReader(
		strings.NewReader("partial_data"),
		&errorReader{err: io.ErrUnexpectedEOF},
	)

	testReader := io.MultiReader(headerBuf, errorReader)

	_, err = ReadMessageFromTunnel(testReader)
	assert.ErrorContains(err, "unexpected EOF", "Should propagate data read errors")
}

func TestReadMessageFromTunnel_MaxDataLength(t *testing.T) {
	assert := assert.New(t)
	data := bytes.Repeat([]byte{0x41}, constants.MaxRespBodyLength)
	msg := NewMessage(1, MessageTypeData, data)

	readMsg, err := ReadMessageFromTunnel(bytes.NewReader(msg.Bytes()))
	assert.NoError(err)
	assert.Len(readMsg.Data, constants.MaxRespBodyLength)
}

// errorReader helper implements io.Reader returning specified error
type errorReader struct{ err error }

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}
