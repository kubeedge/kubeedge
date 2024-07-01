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

func TestExecConnection_CreateConnectMessage(t *testing.T) {
	assert := assert.New(t)

	edgedExecConn := &EdgedExecConnection{
		MessID: 1,
	}
	msg, err := edgedExecConn.CreateConnectMessage()
	assert.NoError(err)

	exceptedData, err := json.Marshal(edgedExecConn)
	assert.NoError(err)
	expectedMessage := NewMessage(edgedExecConn.MessID, MessageTypeExecConnect, exceptedData)

	assert.Equal(expectedMessage, msg)
}

func TestExecConnection_GetMessageID(t *testing.T) {
	assert := assert.New(t)
	edgedExecConn := &EdgedExecConnection{
		MessID: uint64(100),
	}

	stdResult := uint64(100)
	assert.Equal(stdResult, edgedExecConn.MessID)
}

func TestExecConnection_String(t *testing.T) {
	assert := assert.New(t)

	edgedExecConn := &EdgedExecConnection{
		MessID: uint64(100),
	}

	result := edgedExecConn.String()
	stdResult := "EDGE_EXEC_CONNECTOR Message MessageID 100"
	assert.Equal(stdResult, result)
}

func TestExecConnection_CacheTunnelMessage(t *testing.T) {
	assert := assert.New(t)
	edgedExecConn := &EdgedExecConnection{
		ReadChan: make(chan *Message, 1),
	}

	msg := &Message{ConnectID: 100, MessageType: MessageTypeData, Data: []byte("test data")}
	edgedExecConn.CacheTunnelMessage(msg)

	assert.Equal(msg, <-edgedExecConn.ReadChan)
}

func TestExecConnection_CloseReadChannel(t *testing.T) {
	assert := assert.New(t)
	edgedExecConn := &EdgedExecConnection{
		ReadChan: make(chan *Message),
	}

	go func() {
		time.Sleep(1 * time.Second)
		edgedExecConn.CloseReadChannel()
	}()

	_, ok := <-edgedExecConn.ReadChan
	assert.False(ok)
}
