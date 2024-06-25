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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogConnection_CreateConnectMessage(t *testing.T) {
	assert := assert.New(t)

	edgedLogsConn := &EdgedLogsConnection{
		MessID: 1,
	}
	msg, err := edgedLogsConn.CreateConnectMessage()
	assert.NoError(err)

	exceptedData, err := json.Marshal(edgedLogsConn)
	assert.NoError(err)
	expectedMessage := NewMessage(edgedLogsConn.MessID, MessageTypeLogsConnect, exceptedData)

	assert.Equal(expectedMessage, msg)
}

func TestLogConnection_GetMessageID(t *testing.T) {
	assert := assert.New(t)
	edgedLogsConn := &EdgedLogsConnection{
		MessID: uint64(100),
	}

	stdResult := uint64(100)
	assert.Equal(stdResult, edgedLogsConn.MessID)
}

func TestLogConnection_String(t *testing.T) {
	assert := assert.New(t)

	edgedLogsConn := &EdgedLogsConnection{
		MessID: uint64(100),
	}

	result := edgedLogsConn.String()
	stdResult := "EDGE_LOGS_CONNECTOR Message MessageID 100"
	assert.Equal(stdResult, result)
}

func TestLogConnection_CacheTunnelMessage(t *testing.T) {
	assert := assert.New(t)
	edgedLogsConn := &EdgedLogsConnection{
		ReadChan: make(chan *Message, 1),
	}

	msg := &Message{ConnectID: 100, MessageType: MessageTypeData, Data: []byte("test data")}
	edgedLogsConn.CacheTunnelMessage(msg)

	assert.Equal(msg, <-edgedLogsConn.ReadChan)
}

func TestLogConnection_CloseReadChannel(t *testing.T) {
	assert := assert.New(t)
	edgedLogsConn := &EdgedLogsConnection{
		ReadChan: make(chan *Message),
	}

	go func() {
		time.Sleep(1 * time.Second)
		edgedLogsConn.CloseReadChannel()
	}()

	_, ok := <-edgedLogsConn.ReadChan
	assert.False(ok)
}
