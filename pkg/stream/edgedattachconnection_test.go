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

func TestCreateConnectMessage(t *testing.T) {
	assert := assert.New(t)
	edgedAttachConn := &EdgedAttachConnection{
		MessID: 1,
	}

	msg, err := edgedAttachConn.CreateConnectMessage()
	assert.NoError(err)

	expectedData, err := json.Marshal(edgedAttachConn)
	assert.NoError(err)
	expectedMessage := NewMessage(edgedAttachConn.MessID, MessageTypeAttachConnect, expectedData)

	assert.Equal(expectedMessage, msg)
}

func TestGetMessageID(t *testing.T) {
	assert := assert.New(t)

	edgedAttachConn := &EdgedAttachConnection{
		MessID: uint64(100),
	}

	messID := edgedAttachConn.GetMessageID()
	stdResult := uint64(100)

	assert.Equal(messID, stdResult)
}

func TestString(t *testing.T) {
	assert := assert.New(t)

	edgedAttachConn := &EdgedAttachConnection{
		MessID: uint64(100),
	}

	stdResult := "EDGE_ATTACH_CONNECTOR Message MessageID 100"
	result := edgedAttachConn.String()

	assert.Equal(result, stdResult)
}

func TestCacheTunnelMessage(t *testing.T) {
	assert := assert.New(t)
	edgedAttachConn := &EdgedAttachConnection{
		ReadChan: make(chan *Message, 1),
	}

	msg := &Message{ConnectID: 100, MessageType: MessageTypeData, Data: []byte("test data")}
	edgedAttachConn.CacheTunnelMessage(msg)

	assert.Equal(msg, <-edgedAttachConn.ReadChan)
}

func TestCloseReadChannel(t *testing.T) {
	assert := assert.New(t)
	edgedAttachConn := &EdgedAttachConnection{
		ReadChan: make(chan *Message),
	}

	go func() {
		time.Sleep(1 * time.Second)
		edgedAttachConn.CloseReadChannel()
	}()

	_, ok := <-edgedAttachConn.ReadChan
	assert.False(ok)
}
