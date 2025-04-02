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

package flushwriter

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockFlusher implements http.Flusher for testing
type mockFlusher struct {
	flushCalled bool
}

func (m *mockFlusher) Flush() {
	m.flushCalled = true
}

// mockWriterFlusher implements both io.Writer and http.Flusher
type mockWriterFlusher struct {
	*bytes.Buffer
	*mockFlusher
}

func TestFlushWriter_Write(t *testing.T) {
	tests := []struct {
		name          string
		writer        io.Writer
		input         []byte
		expectedN     int
		expectedErr   error
		flushExpected bool
	}{
		{
			name:          "Write with flusher",
			writer:        &mockWriterFlusher{&bytes.Buffer{}, &mockFlusher{}},
			input:         []byte("test data"),
			expectedN:     9,
			expectedErr:   nil,
			flushExpected: true,
		},
		{
			name:          "Write without flusher",
			writer:        &bytes.Buffer{},
			input:         []byte("test data"),
			expectedN:     9,
			expectedErr:   nil,
			flushExpected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := Wrap(tt.writer)
			n, err := fw.Write(tt.input)

			assert.Equal(t, tt.expectedN, n)
			assert.Equal(t, tt.expectedErr, err)

			if tt.flushExpected {
				if mw, ok := tt.writer.(*mockWriterFlusher); ok {
					assert.True(t, mw.flushCalled)
				}
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name          string
		writer        io.Writer
		expectFlusher bool
	}{
		{
			name:          "Wrap writer with flusher",
			writer:        &mockWriterFlusher{&bytes.Buffer{}, &mockFlusher{}},
			expectFlusher: true,
		},
		{
			name:          "Wrap writer without flusher",
			writer:        &bytes.Buffer{},
			expectFlusher: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := Wrap(tt.writer)
			fw, ok := wrapped.(*FlushWriter)
			assert.True(t, ok)

			if tt.expectFlusher {
				assert.NotNil(t, fw.flusher)
			} else {
				assert.Nil(t, fw.flusher)
			}
			assert.Equal(t, tt.writer, fw.writer)
		})
	}
}

func TestFlushWriter_WriteError(t *testing.T) {
	// Create a mock writer that returns an error
	errWriter := &struct {
		io.Writer
	}{}
	errWriter.Writer = writerFunc(func(p []byte) (int, error) {
		return 0, errors.New("mock write error")
	})

	fw := &FlushWriter{
		writer: errWriter,
	}

	_, err := fw.Write([]byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock write error")
}

// writerFunc implements io.Writer
type writerFunc func(p []byte) (n int, err error)

func (f writerFunc) Write(p []byte) (n int, err error) {
	return f(p)
}
