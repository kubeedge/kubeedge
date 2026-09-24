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

package conn

import (
	"io"
	"testing"
	"time"

	"github.com/lucas-clemente/quic-go"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/keeper"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/smgr"
)

type fakeQuicSession struct {
	quic.Session
	stream quic.Stream
}

func (s *fakeQuicSession) OpenStreamSync() (quic.Stream, error) {
	return s.stream, nil
}

type fakeQuicStream struct {
	quic.Stream
}

func (s *fakeQuicStream) StreamID() quic.StreamID {
	return quic.StreamID(1)
}

func (s *fakeQuicStream) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (s *fakeQuicStream) Write(p []byte) (int, error) {
	return len(p), nil
}

func (s *fakeQuicStream) Close() error {
	return nil
}

func (s *fakeQuicStream) CancelWrite(quic.ErrorCode) error {
	return nil
}

func (s *fakeQuicStream) CancelRead(quic.ErrorCode) error {
	return nil
}

func (s *fakeQuicStream) SetWriteDeadline(time.Time) error {
	return nil
}

func TestQuicConnectionWriteMessageSyncReturnsWaitResponseError(t *testing.T) {
	session := &fakeQuicSession{stream: &fakeQuicStream{}}
	connection := &QuicConnection{
		session:       smgr.Session{Sess: session},
		streamManager: smgr.NewStreamManager(smgr.NumStreamsMax, autoFree, session),
		syncKeeper:    keeper.NewSyncKeeper(),
		writeDeadline: time.Now().Add(time.Millisecond),
	}

	msg := model.NewMessage("")
	response, err := connection.WriteMessageSync(msg)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if response != nil {
		t.Fatalf("expected nil response on error, got %v", response)
	}
	expected := "wait response timeout, message id:" + msg.GetID()
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}
