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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewReader(t *testing.T) {
	mockReader := &bytes.Buffer{}
	reader := NewReader(mockReader)

	assert.NotNil(t, reader)
	assert.Equal(t, mockReader, reader.reader)
}

func TestReader_Read_NilReader(t *testing.T) {
	reader := &Reader{reader: nil}
	data, err := reader.Read()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad io reader")
	assert.Nil(t, data)
}

func TestReader_Read_HeaderReadError(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	reader := NewReader(buffer)

	data, err := reader.Read()

	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, data)
}

func TestReader_Read_Success(t *testing.T) {
	testPayload := []byte("test payload")

	header := &PackageHeader{
		Version:     1,
		PackageType: Message,
		Flags:       0,
		PayloadLen:  uint32(len(testPayload)),
	}

	headerBuffer := make([]byte, HeaderSize)

	headerBuffer[VersionOffset] = byte(header.Version >> 24)
	headerBuffer[VersionOffset+1] = byte(header.Version >> 16)
	headerBuffer[VersionOffset+2] = byte(header.Version >> 8)
	headerBuffer[VersionOffset+3] = byte(header.Version)

	headerBuffer[PackageTypeOffset] = byte(header.PackageType)
	headerBuffer[FlagsOffset] = byte(header.Flags)

	headerBuffer[PayloadLenOffset] = byte(header.PayloadLen >> 24)
	headerBuffer[PayloadLenOffset+1] = byte(header.PayloadLen >> 16)
	headerBuffer[PayloadLenOffset+2] = byte(header.PayloadLen >> 8)
	headerBuffer[PayloadLenOffset+3] = byte(header.PayloadLen)

	buffer := bytes.NewBuffer(append(headerBuffer, testPayload...))

	reader := NewReader(buffer)
	data, err := reader.Read()

	assert.NoError(t, err)
	assert.Equal(t, testPayload, data)
}

func TestReader_Read_PayloadReadError(t *testing.T) {
	header := &PackageHeader{
		Version:     1,
		PackageType: Message,
		Flags:       0,
		PayloadLen:  10,
	}

	headerBuffer := make([]byte, HeaderSize)

	headerBuffer[VersionOffset] = byte(header.Version >> 24)
	headerBuffer[VersionOffset+1] = byte(header.Version >> 16)
	headerBuffer[VersionOffset+2] = byte(header.Version >> 8)
	headerBuffer[VersionOffset+3] = byte(header.Version)

	headerBuffer[PackageTypeOffset] = byte(header.PackageType)
	headerBuffer[FlagsOffset] = byte(header.Flags)

	headerBuffer[PayloadLenOffset] = byte(header.PayloadLen >> 24)
	headerBuffer[PayloadLenOffset+1] = byte(header.PayloadLen >> 16)
	headerBuffer[PayloadLenOffset+2] = byte(header.PayloadLen >> 8)
	headerBuffer[PayloadLenOffset+3] = byte(header.PayloadLen)

	buffer := bytes.NewBuffer(headerBuffer)

	reader := NewReader(buffer)
	data, err := reader.Read()

	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, data)
}
