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
package packer

import (
	"bytes"
	"errors"
	"testing"
)

type mockWriter struct {
	fail bool
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	if m.fail {
		return 0, errors.New("mock write failure")
	}
	return len(p), nil
}

func TestWriter_Write(t *testing.T) {
	// Create a mock header
	header := NewPackageHeader(Message)
	header.SetPayloadLen(9)
	var headerBuffer []byte
	header.Pack(&headerBuffer)

	var buf bytes.Buffer
	writer := NewWriter(&buf)
	data := []byte("test data")

	// Successful write
	n, err := writer.Write(data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}

	// Writer with nil io.Writer
	nilWriter := NewWriter(nil)
	_, err = nilWriter.Write(data)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	failingWriter := NewWriter(&mockWriter{fail: true})
	_, err = failingWriter.Write(data)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
