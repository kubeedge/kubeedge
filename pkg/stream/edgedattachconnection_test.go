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
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

func setupEdgedAttachConn(t *testing.T) *EdgedAttachConnection {
	t.Helper()
	return &EdgedAttachConnection{
		ReadChan: make(chan *Message, 2),
		MessID:   100,
		Stop:     make(chan struct{}, 1),
	}
}

func setupMockConn(t *testing.T, readData []byte, readErr, writeErr error) *MockConn {
	t.Helper()
	return &MockConn{
		ReadData:   readData,
		ReadError:  readErr,
		WriteError: writeErr,
	}
}

func setupMockTunneler(t *testing.T, writeErr error) *MockSafeWriteTunneler {
	t.Helper()
	return &MockSafeWriteTunneler{
		WriteError: writeErr,
	}
}

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

func TestCleanChannel(t *testing.T) {
	assert := assert.New(t)
	edgedAttachConn := &EdgedAttachConnection{
		Stop: make(chan struct{}, 3),
	}

	edgedAttachConn.Stop <- struct{}{}
	edgedAttachConn.Stop <- struct{}{}
	edgedAttachConn.Stop <- struct{}{}

	assert.Equal(3, len(edgedAttachConn.Stop))

	edgedAttachConn.CleanChannel()

	assert.Equal(0, len(edgedAttachConn.Stop))
}

type MockConn struct {
	ReadData       []byte
	ReadError      error
	WrittenData    []byte
	CloseCallCount int
	WriteError     error
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	if m.ReadError != nil {
		return 0, m.ReadError
	}
	if len(m.ReadData) == 0 {
		return 0, io.EOF
	}
	n = copy(b, m.ReadData)
	m.ReadData = m.ReadData[n:]
	return n, nil
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	if m.WriteError != nil {
		return 0, m.WriteError
	}
	m.WrittenData = append(m.WrittenData, b...)
	return len(b), nil
}

func (m *MockConn) Close() error {
	m.CloseCallCount++
	return nil
}

func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

// Mock SafeWriteTunneler for testing
type MockSafeWriteTunneler struct {
	WrittenMessages []*Message
	WriteError      error
	CloseCallCount  int
}

func (m *MockSafeWriteTunneler) WriteMessage(msg *Message) error {
	if m.WriteError != nil {
		return m.WriteError
	}
	m.WrittenMessages = append(m.WrittenMessages, msg)
	return nil
}

func (m *MockSafeWriteTunneler) Close() error {
	m.CloseCallCount++
	return nil
}

// These methods are not used in our tests but needed to satisfy the interface
func (m *MockSafeWriteTunneler) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return nil
}

func (m *MockSafeWriteTunneler) NextReader() (messageType int, r io.Reader, err error) {
	return 0, nil, nil
}

func TestReceiveFromCloudStream(t *testing.T) {
	patchKlog := gomonkey.ApplyFuncSeq(klog.Errorf, []gomonkey.OutputCell{
		{Values: gomonkey.Params{}, Times: 100},
	})
	defer patchKlog.Reset()

	t.Run("MessageTypeData", func(t *testing.T) {
		assert := assert.New(t)

		mockConn := setupMockConn(t, nil, nil, nil)
		edgedAttachConn := setupEdgedAttachConn(t)

		testData := []byte("test data")
		dataMsg := NewMessage(edgedAttachConn.MessID, MessageTypeData, testData)

		stopChan := make(chan struct{}, 1)
		go edgedAttachConn.receiveFromCloudStream(mockConn, stopChan)

		edgedAttachConn.ReadChan <- dataMsg

		time.Sleep(10 * time.Millisecond)

		assert.Equal(testData, mockConn.WrittenData)

		close(edgedAttachConn.ReadChan)
	})

	t.Run("MessageTypeRemoveConnect", func(t *testing.T) {
		mockConn := setupMockConn(t, nil, nil, nil)
		edgedAttachConn := setupEdgedAttachConn(t)

		stopChan := make(chan struct{}, 1)
		go edgedAttachConn.receiveFromCloudStream(mockConn, stopChan)

		removeMsg := NewMessage(edgedAttachConn.MessID, MessageTypeRemoveConnect, nil)
		edgedAttachConn.ReadChan <- removeMsg

		select {
		case <-stopChan:
		case <-time.After(10 * time.Millisecond):
			t.Error("Did not receive stop signal")
		}

		close(edgedAttachConn.ReadChan)
	})

	t.Run("Write error", func(t *testing.T) {
		mockConn := setupMockConn(t, nil, nil, errors.New("write error"))
		edgedAttachConn := setupEdgedAttachConn(t)

		stopChan := make(chan struct{}, 1)
		go edgedAttachConn.receiveFromCloudStream(mockConn, stopChan)

		dataMsg := NewMessage(edgedAttachConn.MessID, MessageTypeData, []byte("test data"))
		edgedAttachConn.ReadChan <- dataMsg

		time.Sleep(10 * time.Millisecond)

		close(edgedAttachConn.ReadChan)
	})
}

func TestWrite2CloudStream(t *testing.T) {
	patchKlog := gomonkey.ApplyFuncSeq(klog.Errorf, []gomonkey.OutputCell{
		{Values: gomonkey.Params{}, Times: 100},
	})
	defer patchKlog.Reset()

	t.Run("Normal operation", func(t *testing.T) {
		assert := assert.New(t)

		mockConn := setupMockConn(t, []byte("test data"), nil, nil)
		mockTunnel := setupMockTunneler(t, nil)
		edgedAttachConn := setupEdgedAttachConn(t)

		stopChan := make(chan struct{}, 1)
		done := make(chan struct{})

		go func() {
			edgedAttachConn.write2CloudStream(mockTunnel, mockConn, stopChan)
			close(done)
		}()

		time.Sleep(10 * time.Millisecond)

		assert.Equal(1, len(mockTunnel.WrittenMessages))
		assert.Equal([]byte("test data"), mockTunnel.WrittenMessages[0].Data)

		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Error("write2CloudStream didn't complete")
		}
	})

	t.Run("Read error", func(t *testing.T) {
		mockConn := setupMockConn(t, nil, errors.New("read error"), nil)
		mockTunnel := setupMockTunneler(t, nil)
		edgedAttachConn := setupEdgedAttachConn(t)

		stopChan := make(chan struct{}, 1)

		go edgedAttachConn.write2CloudStream(mockTunnel, mockConn, stopChan)

		select {
		case <-stopChan:
		case <-time.After(10 * time.Millisecond):
			t.Error("Did not receive stop signal on read error")
		}
	})

	t.Run("Write error", func(t *testing.T) {
		mockConn := setupMockConn(t, []byte("test data"), nil, nil)
		mockTunnel := setupMockTunneler(t, errors.New("write error"))
		edgedAttachConn := setupEdgedAttachConn(t)

		stopChan := make(chan struct{}, 1)

		go edgedAttachConn.write2CloudStream(mockTunnel, mockConn, stopChan)

		select {
		case <-stopChan:
		case <-time.After(10 * time.Millisecond):
			t.Error("Did not receive stop signal on write error")
		}
	})
}
