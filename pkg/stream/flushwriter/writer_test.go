/*
Copyright 2022 The KubeEdge Authors.

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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ErrorWriter struct {
	io.Writer
}

func (w *ErrorWriter) Write(p []byte) (int, error) {
	return 0, errors.New("raise an error")
}

func TestFlushWriter_Write(t *testing.T) {
	tests := []struct {
		name    string
		flusher http.Flusher
		writer  io.Writer
		p       []byte
		wantN   int
		wantErr bool
	}{
		{
			name:    "write error",
			writer:  &ErrorWriter{},
			p:       []byte("test"),
			wantN:   0,
			wantErr: true,
		},
		{
			name:   "set writer",
			writer: bytes.NewBuffer(make([]byte, 1024)),
			p:      []byte("test"),
			wantN:  4,
		},
		{
			name:    "set writer and flusher",
			flusher: httptest.NewRecorder(),
			writer:  bytes.NewBuffer(make([]byte, 1024)),
			p:       []byte("test"),
			wantN:   4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FlushWriter{
				flusher: tt.flusher,
				writer:  tt.writer,
			}
			gotN, err := f.Write(tt.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("FlushWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("FlushWriter.Write() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name            string
		argW            io.Writer
		wantFlushWriter bool
	}{
		{
			name:            "base",
			argW:            bytes.NewBuffer(make([]byte, 1024)),
			wantFlushWriter: true,
		},
		{
			name:            "set flusher",
			argW:            httptest.NewRecorder(),
			wantFlushWriter: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrap(tt.argW)
			fmt.Printf("%+v\n", got)
			if _, ok := got.(*FlushWriter); ok != tt.wantFlushWriter {
				t.Errorf("Wrap() = %v, want %v", ok, tt.wantFlushWriter)
			}
		})
	}
}
