/*
Copyright 2019 The KubeEdge Authors.

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
	"fmt"
	"io"

	"k8s.io/klog"
)

type Reader struct {
	reader io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{reader: r}
}

// Read message raw data from reader
// steps:
// 1)read the package header
// 2)unpack the package header and get the payload length
// 3)read the payload
func (r *Reader) Read() ([]byte, error) {
	if r.reader == nil {
		klog.Error("bad io reader")
		return nil, fmt.Errorf("bad io reader")
	}

	headerBuffer := make([]byte, HeaderSize)
	_, err := io.ReadFull(r.reader, headerBuffer)
	if err != nil {
		if err != io.EOF {
			klog.Error("failed to read package header from buffer")
		}
		return nil, err
	}

	header := PackageHeader{}
	header.Unpack(headerBuffer)

	payloadBuffer := make([]byte, header.PayloadLen)
	_, err = io.ReadFull(r.reader, payloadBuffer)
	if err != nil {
		if err != io.EOF {
			klog.Error("failed to read payload from buffer")
		}
		return nil, err
	}

	return payloadBuffer, nil
}
