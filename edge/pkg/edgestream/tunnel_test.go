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
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

type mockEdgedConnection struct {
	cacheCalled    bool
	cleanCalled    bool
	closedReadChan bool
}

func (m *mockEdgedConnection) CacheTunnelMessage(msg *stream.Message) {
	m.cacheCalled = true
}

func (m *mockEdgedConnection) CreateConnectMessage() (*stream.Message, error) {
	return nil, nil
}

func (m *mockEdgedConnection) GetMessageID() uint64 {
	return 0
}

func (m *mockEdgedConnection) String() string {
	return "mockEdgedConnection"
}

func (m *mockEdgedConnection) CleanChannel() {
	m.cleanCalled = true
}

func (m *mockEdgedConnection) CloseReadChannel() {
	m.closedReadChan = true
}

func (m *mockEdgedConnection) Serve(tunnel stream.SafeWriteTunneler) error {
	return nil
}

type mockTunnel struct {
	closed bool
}

func (m *mockTunnel) WriteMessage(_ *stream.Message) error {
	return nil
}

func (m *mockTunnel) WriteControl(_ int, _ []byte, _ time.Time) error {
	return nil
}

func (m *mockTunnel) Close() error {
	m.closed = true
	return nil
}

func (m *mockTunnel) NextReader() (messageType int, reader io.Reader, err error) {
	return 0, nil, nil
}

func TestAddAndGetLocalConnection(t *testing.T) {
	session := &TunnelSession{
		localCons: make(map[uint64]stream.EdgedConnection),
	}
	mockConn := &mockEdgedConnection{}

	session.AddLocalConnection(1, mockConn)

	conn, ok := session.GetLocalConnection(1)
	assert.True(t, ok)
	assert.Equal(t, mockConn, conn)

	_, ok = session.GetLocalConnection(999)
	assert.False(t, ok)
}

func TestDeleteLocalConnection(t *testing.T) {
	session := &TunnelSession{
		localCons: make(map[uint64]stream.EdgedConnection),
	}
	mockConn := &mockEdgedConnection{}
	session.AddLocalConnection(1, mockConn)

	session.DeleteLocalConnection(1)

	_, ok := session.GetLocalConnection(1)
	assert.False(t, ok)
	assert.True(t, mockConn.cleanCalled)
	assert.True(t, mockConn.closedReadChan)

	// delete nonexistent, should not panic
	assert.NotPanics(t, func() {
		session.DeleteLocalConnection(999)
	})
}

func TestCloseTunnelSession(t *testing.T) {
	mockTun := &mockTunnel{}
	session := &TunnelSession{
		Tunnel:    mockTun,
	}

	session.Close()
	assert.True(t, mockTun.closed)
	assert.True(t, session.closed)

	// Idempotent close
	assert.NotPanics(t, func() {
		session.Close()
	})
}

func TestWriteToLocalConnection(t *testing.T) {
	session := &TunnelSession{
		localCons: make(map[uint64]stream.EdgedConnection),
	}
	mockConn := &mockEdgedConnection{}
	session.AddLocalConnection(1, mockConn)

	msg := &stream.Message{
		ConnectID: 1,
	}
	session.WriteToLocalConnection(msg)
	assert.True(t, mockConn.cacheCalled)

	// Write to nonexistent connection
	msg2 := &stream.Message{
		ConnectID: 999,
	}
	assert.NotPanics(t, func() {
		session.WriteToLocalConnection(msg2)
	})
}

func TestServeConnectionPanicsOnUnknownMessage(t *testing.T) {
	session := &TunnelSession{
		localCons: make(map[uint64]stream.EdgedConnection),
	}
	msg := &stream.Message{
		MessageType: stream.MessageTypeData,
	}

	assert.Panics(t, func() {
		session.ServeConnection(msg)
	})
}

func TestConcurrentMapOperations(t *testing.T) {
	session := &TunnelSession{
		localCons: make(map[uint64]stream.EdgedConnection),
	}

	var wg sync.WaitGroup
	workers := 10
	wg.Add(workers * 2)

	// Concurrent writes
	for i := 0; i < workers; i++ {
		go func(id uint64) {
			defer wg.Done()
			session.AddLocalConnection(id, &mockEdgedConnection{})
		}(uint64(i))
	}

	// Concurrent reads/deletes
	for i := 0; i < workers; i++ {
		go func(id uint64) {
			defer wg.Done()
			session.GetLocalConnection(id)
			session.DeleteLocalConnection(id)
		}(uint64(i))
	}

	wg.Wait()
}
