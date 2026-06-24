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

package edgestream

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

func TestTunnelSession_ServeConnectionUnsupportedMessageType(t *testing.T) {
	session := &TunnelSession{}

	assert.NotPanics(t, func() {
		session.ServeConnection(&stream.Message{
			ConnectID:   1,
			MessageType: stream.MessageType(99),
		})
	})
}

func TestTunnelSession_RouteMessageUnsupportedMessageType(t *testing.T) {
	assert := assert.New(t)
	con := &testEdgedConnection{}
	session := &TunnelSession{
		localCons: map[uint64]stream.EdgedConnection{
			1: con,
		},
	}

	err := session.routeMessage(&stream.Message{
		ConnectID:   1,
		MessageType: stream.MessageType(99),
	})

	assert.NoError(err)
	assert.Empty(con.cachedMessages)
}

type testEdgedConnection struct {
	cachedMessages []*stream.Message
}

func (c *testEdgedConnection) CreateConnectMessage() (*stream.Message, error) {
	return nil, nil
}

func (c *testEdgedConnection) Serve(stream.SafeWriteTunneler) error {
	return nil
}

func (c *testEdgedConnection) CacheTunnelMessage(msg *stream.Message) {
	c.cachedMessages = append(c.cachedMessages, msg)
}

func (c *testEdgedConnection) GetMessageID() uint64 {
	return 0
}

func (c *testEdgedConnection) CloseReadChannel() {}

func (c *testEdgedConnection) CleanChannel() {}

func (c *testEdgedConnection) String() string {
	return fmt.Sprintf("%d", len(c.cachedMessages))
}
