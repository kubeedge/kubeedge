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

func TestMetricsConnection_CreateConnectMessage(t *testing.T) {
	assert := assert.New(t)
	edgedMetricsConn := &EdgedMetricsConnection{
		MessID: 1,
	}

	msg, err := edgedMetricsConn.CreateConnectMessage()
	assert.NoError(err)

	expectedData, err := json.Marshal(edgedMetricsConn)
	assert.NoError(err)
	expectedMessage := NewMessage(edgedMetricsConn.MessID, MessageTypeMetricConnect, expectedData)

	assert.Equal(expectedMessage, msg)
}

func TestMetricsConnection_GetMessageID(t *testing.T) {
	assert := assert.New(t)

	edgedMetricsConn := &EdgedMetricsConnection{
		MessID: uint64(100),
	}

	messID := edgedMetricsConn.GetMessageID()
	stdResult := uint64(100)

	assert.Equal(messID, stdResult)
}

func TestMetricsConnection_String(t *testing.T) {
	assert := assert.New(t)

	edgedMetricsConn := &EdgedMetricsConnection{
		MessID: uint64(100),
	}

	stdResult := "EDGE_METRICS_CONNECTOR Message MessageID 100"
	result := edgedMetricsConn.String()

	assert.Equal(result, stdResult)
}

func TestMetricsConnection_CacheTunnelMessage(t *testing.T) {
	assert := assert.New(t)
	edgedMetricsConn := &EdgedMetricsConnection{
		ReadChan: make(chan *Message, 1),
	}

	msg := &Message{ConnectID: 100, MessageType: MessageTypeData, Data: []byte("test data")}
	edgedMetricsConn.CacheTunnelMessage(msg)

	assert.Equal(msg, <-edgedMetricsConn.ReadChan)
}

func TestMetricsConnection_CloseReadChannel(t *testing.T) {
	assert := assert.New(t)
	edgedMetricsConn := &EdgedMetricsConnection{
		ReadChan: make(chan *Message),
	}

	go func() {
		time.Sleep(1 * time.Second)
		edgedMetricsConn.CloseReadChannel()
	}()

	_, ok := <-edgedMetricsConn.ReadChan
	assert.False(ok)
}
