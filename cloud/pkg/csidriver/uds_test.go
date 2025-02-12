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

package csidriver

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createTempSocketPath(t *testing.T) string {
	dir := t.TempDir()
	return filepath.Join(dir, "test.sock")
}

func setupSocket(_ *testing.T, path string) (net.Listener, error) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

func TestNewUnixDomainSocket(t *testing.T) {
	assert := assert.New(t)

	us := NewUnixDomainSocket("/tmp/test.sock")
	assert.NotNil(us)
	assert.Equal("/tmp/test.sock", us.filename)
	assert.Equal(DefaultBufferSize, us.buffersize)

	us = NewUnixDomainSocket("/tmp/test.sock", 2048)
	assert.NotNil(us)
	assert.Equal("/tmp/test.sock", us.filename)
	assert.Equal(2048, us.buffersize)
}

func TestConnect(t *testing.T) {
	assert := assert.New(t)

	us := NewUnixDomainSocket("invalid://endpoint")
	conn, err := us.Connect()
	assert.Error(err)
	assert.Nil(conn)

	socketPath := createTempSocketPath(t)

	listener, err := setupSocket(t, socketPath)
	if err != nil {
		t.Fatalf("Failed to setup socket: %v", err)
	}
	defer listener.Close()

	connected := make(chan struct{})
	go func() {
		defer close(connected)
		conn, err := listener.Accept()
		if err == nil && conn != nil {
			conn.Close()
		}
	}()

	time.Sleep(100 * time.Millisecond)

	us = NewUnixDomainSocket("unix://" + socketPath)
	conn, err = us.Connect()
	assert.NoError(err)
	assert.NotNil(conn)
	if conn != nil {
		conn.Close()
	}

	<-connected

	us = NewUnixDomainSocket("unix:///nonexistent/path/sock")
	conn, err = us.Connect()
	assert.Error(err)
	assert.Nil(conn)
}

func TestSend(t *testing.T) {
	assert := assert.New(t)
	socketPath := createTempSocketPath(t)

	listener, err := setupSocket(t, socketPath)
	if err != nil {
		t.Fatalf("Failed to setup socket: %v", err)
	}
	defer listener.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		if _, err := conn.Read(buf); err != nil {
			return
		}

		if _, err := conn.Write([]byte("response message")); err != nil {
			return
		}
	}()

	us := NewUnixDomainSocket("unix://" + socketPath)
	conn, err := us.Connect()
	assert.NoError(err)
	assert.NotNil(conn)

	if conn != nil {
		response, err := us.Send(conn, "test message")
		assert.NoError(err)
		assert.Equal("response message", response)
		conn.Close()
	}

	<-done
}

func TestSendWithBufferSizeLimit(t *testing.T) {
	assert := assert.New(t)
	socketPath := createTempSocketPath(t)

	listener, err := setupSocket(t, socketPath)
	if err != nil {
		t.Fatalf("Failed to setup socket: %v", err)
	}
	defer listener.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		if _, err := conn.Write([]byte("this is a response message that exceeds the small buffer size")); err != nil {
			return
		}
	}()

	us := NewUnixDomainSocket("unix://"+socketPath, 10)
	conn, err := us.Connect()
	assert.NoError(err)
	assert.NotNil(conn)

	if conn != nil {
		response, err := us.Send(conn, "test")
		assert.NoError(err)
		assert.Len(response, 10)
		conn.Close()
	}

	<-done
}
