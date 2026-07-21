/*
Copyright 2026 The KubeEdge Authors.

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
)

func TestEdgedExecConnection_GetMessageID(t *testing.T) {
	expectedID := uint64(12345)
	conn := &EdgedExecConnection{
		MessID: expectedID,
	}

	if got := conn.GetMessageID(); got != expectedID {
		t.Errorf("EdgedExecConnection.GetMessageID() = %v, want %v", got, expectedID)
	}
}

func TestEdgedExecConnection_String(t *testing.T) {
	conn := &EdgedExecConnection{
		MessID: 54321,
	}

	expectedStr := "EDGE_EXEC_CONNECTOR Message MessageID 54321"
	if got := conn.String(); got != expectedStr {
		t.Errorf("EdgedExecConnection.String() = %v, want %v", got, expectedStr)
	}
}

func TestEdgedExecConnection_CreateConnectMessage(t *testing.T) {
	conn := &EdgedExecConnection{
		MessID: 999,
		Method: "GET",
	}

	msg, err := conn.CreateConnectMessage()
	if err != nil {
		t.Errorf("EdgedExecConnection.CreateConnectMessage() unexpected error: %v", err)
	}
	
	if msg.ConnectID != 999 {
		t.Errorf("EdgedExecConnection.CreateConnectMessage() message ID = %v, want %v", msg.ConnectID, 999)
	}

	if msg.MessageType != MessageTypeExecConnect {
		t.Errorf("EdgedExecConnection.CreateConnectMessage() message type = %v, want %v", msg.MessageType, MessageTypeExecConnect)
	}

	var unmarshaledConn EdgedExecConnection
	if err := json.Unmarshal(msg.Data, &unmarshaledConn); err != nil {
		t.Errorf("EdgedExecConnection.CreateConnectMessage() unexpected unmarshal error: %v", err)
	}

	if unmarshaledConn.Method != "GET" {
		t.Errorf("EdgedExecConnection.CreateConnectMessage() unmarshaled method = %v, want %v", unmarshaledConn.Method, "GET")
	}
}
