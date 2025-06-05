/*
Copyright 2025 The KubeEdge Authors.

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
	"io"
	"net"
	"time"
)

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

func (m *MockConn) LocalAddr() net.Addr { return nil }

func (m *MockConn) RemoteAddr() net.Addr { return nil }

func (m *MockConn) SetDeadline(t time.Time) error { return nil }

func (m *MockConn) SetReadDeadline(t time.Time) error { return nil }

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

type MockTunneler struct {
	Messages    []*Message
	WriteErr    error
	ControlData []byte
	ControlType int
	ControlErr  error
	ReaderType  int
	ReaderData  []byte
	ReaderErr   error
	CloseErr    error
	Closed      bool
}

func (m *MockTunneler) WriteMessage(msg *Message) error {
	if m.WriteErr != nil {
		return m.WriteErr
	}
	m.Messages = append(m.Messages, msg)
	return nil
}

func (m *MockTunneler) WriteControl(messageType int, data []byte, deadline time.Time) error {
	m.ControlType = messageType
	m.ControlData = data
	return m.ControlErr
}

func (m *MockTunneler) NextReader() (messageType int, r io.Reader, err error) {
	if m.ReaderErr != nil {
		return 0, nil, m.ReaderErr
	}
	return m.ReaderType, io.NopCloser(bytes.NewReader(m.ReaderData)), nil
}

func (m *MockTunneler) Close() error {
	m.Closed = true
	return m.CloseErr
}
