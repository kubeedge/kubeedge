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

package io

import (
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/conn"
)

// CloudHubIO handle the IO operation from connection
type CloudHubIO interface {
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	ReadData(*model.Message) (int, error)
	WriteData(*model.Message) error
	Close() error
}

// JSONIO address the json data from connection
type JSONIO struct {
	Connection conn.Connection
}

// SetReadDeadline set read operation dead line
func (io *JSONIO) SetReadDeadline(time time.Time) error {
	return io.Connection.SetReadDeadline(time)
}

// SetWriteDeadline set write operation dead line
func (io *JSONIO) SetWriteDeadline(time time.Time) error {
	return io.Connection.SetWriteDeadline(time)
}

// ReadData read data from connection
func (io *JSONIO) ReadData(msg *model.Message) (int, error) {
	return 0, io.Connection.ReadMessage(msg)
}

// WriteData write data to connection
func (io *JSONIO) WriteData(msg *model.Message) error {
	err := io.Connection.WriteMessageAsync(msg)
	if err != nil {
		return err
	}
	return nil
}

// Close close the IO operation
func (io *JSONIO) Close() error {
	return io.Connection.Close()
}
