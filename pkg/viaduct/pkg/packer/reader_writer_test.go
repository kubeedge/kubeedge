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
	"strings"
	"testing"
)

func buildTestFrame(payload []byte) []byte {
	header := NewPackageHeader(Message)
	header.SetPayloadLen(uint32(len(payload)))

	var headerBuffer []byte
	header.Pack(&headerBuffer)

	var frame bytes.Buffer
	frame.Write(headerBuffer)
	frame.Write(payload)
	return frame.Bytes()
}

func buildTestHeaderWithPayloadLen(payloadLen uint32) []byte {
	header := NewPackageHeader(Message)
	header.SetPayloadLen(payloadLen)

	var headerBuffer []byte
	header.Pack(&headerBuffer)
	return headerBuffer
}

func TestReaderReadRejectsPayloadLargerThanLimit(t *testing.T) {
	frame := buildTestHeaderWithPayloadLen(MaxPayloadLen + 1)

	_, err := NewReader(bytes.NewReader(frame)).Read()
	if err == nil {
		t.Fatalf("expected error for payload larger than limit")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("expected max payload error, got %v", err)
	}
}

func TestReaderReadAllowsPayloadWithinLimit(t *testing.T) {
	payload := []byte("hello")
	frame := buildTestFrame(payload)

	got, err := NewReader(bytes.NewReader(frame)).Read()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("expected payload %q, got %q", payload, got)
	}
}

func TestWriterWriteRejectsPayloadLargerThanLimit(t *testing.T) {
	payload := make([]byte, int(MaxPayloadLen)+1)

	var buf bytes.Buffer
	n, err := NewWriter(&buf).Write(payload)
	if err == nil {
		t.Fatalf("expected error for payload larger than limit")
	}
	if n != 0 {
		t.Fatalf("expected 0 bytes written, got %d", n)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected nothing written, got %d bytes", buf.Len())
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("expected max payload error, got %v", err)
	}
}

func TestWriterWriteAllowsPayloadWithinLimit(t *testing.T) {
	payload := []byte("hello")

	var buf bytes.Buffer
	n, err := NewWriter(&buf).Write(payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n != len(payload) {
		t.Fatalf("expected %d bytes written, got %d", len(payload), n)
	}

	got, err := NewReader(&buf).Read()
	if err != nil {
		t.Fatalf("expected no error reading written frame, got %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("expected payload %q, got %q", payload, got)
	}
}
