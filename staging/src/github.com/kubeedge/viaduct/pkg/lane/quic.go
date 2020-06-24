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

package lane

import (
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/packer"
	"github.com/kubeedge/viaduct/pkg/translator"
	"github.com/lucas-clemente/quic-go"
)

type QuicLane struct {
	writeDeadline time.Time
	readDeadline  time.Time
	stream        quic.Stream
}

func NewQuicLane(van interface{}) *QuicLane {
	if stream, ok := van.(quic.Stream); ok {
		return &QuicLane{stream: stream}
	}
	klog.Error("oops! bad type of van")
	return nil
}

func (l *QuicLane) ReadMessage(msg *model.Message) error {
	rawData, err := packer.NewReader(l.stream).Read()
	if err != nil {
		return err
	}

	err = translator.NewTran().Decode(rawData, msg)
	if err != nil {
		klog.Error("failed to decode message")
		return err
	}

	return nil
}

func (l *QuicLane) WriteMessage(msg *model.Message) error {
	rawData, err := translator.NewTran().Encode(msg)
	if err != nil {
		klog.Error("failed to encode message")
		return err
	}

	_, err = packer.NewWriter(l.stream).Write(rawData)
	return err
}

func (l *QuicLane) Read(raw []byte) (int, error) {
	return l.stream.Read(raw)
}

func (l *QuicLane) Write(raw []byte) (int, error) {
	return l.stream.Write(raw)
}

func (l *QuicLane) SetReadDeadline(t time.Time) error {
	l.readDeadline = t
	return l.stream.SetReadDeadline(t)
}

func (l *QuicLane) SetWriteDeadline(t time.Time) error {
	l.writeDeadline = t
	return l.stream.SetWriteDeadline(t)
}
