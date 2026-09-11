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

package packer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return false }

func TestSilentReadErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"EOF", io.EOF, true},
		{"wrapped EOF", fmt.Errorf("read: %w", io.EOF), true},
		{"deadline timeout", timeoutErr{}, true},
		{"wrapped timeout", fmt.Errorf("read: %w", timeoutErr{}), true},
		{"unexpected EOF", io.ErrUnexpectedEOF, false},
		{"other", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := silentReadErr(tc.err); got != tc.want {
				t.Fatalf("silentReadErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestReadReturnsTruncatedHeaderAndPayloadErrors verifies that a stream that
// ends mid-header or mid-payload surfaces io.ErrUnexpectedEOF (a genuine
// transport error, not a silent one) from Read.
func TestReadReturnsTruncatedHeaderAndPayloadErrors(t *testing.T) {
	var packed bytes.Buffer
	if _, err := NewWriter(&packed).Write([]byte("hello")); err != nil {
		t.Fatalf("pack: %v", err)
	}
	full := packed.Bytes()

	t.Run("truncated header", func(t *testing.T) {
		_, err := NewReader(bytes.NewReader(full[:1])).Read()
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("got %v, want io.ErrUnexpectedEOF", err)
		}
	})
	t.Run("truncated payload", func(t *testing.T) {
		_, err := NewReader(bytes.NewReader(full[:len(full)-2])).Read()
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("got %v, want io.ErrUnexpectedEOF", err)
		}
	})
	t.Run("clean EOF before header", func(t *testing.T) {
		_, err := NewReader(bytes.NewReader(nil)).Read()
		if !errors.Is(err, io.EOF) {
			t.Fatalf("got %v, want io.EOF", err)
		}
	})
	t.Run("intact", func(t *testing.T) {
		got, err := NewReader(bytes.NewReader(full)).Read()
		if err != nil || string(got) != "hello" {
			t.Fatalf("got (%q, %v), want (\"hello\", nil)", got, err)
		}
	})
}
