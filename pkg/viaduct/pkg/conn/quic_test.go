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
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/mocks"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/lane"
)

// newTestQuicConn builds a connection over a mocked session. No stream is
// registered with the stream manager, so the Destroy call inside Close is a
// no-op here and the test stays focused on the teardown of the connection.
func newTestQuicConn(t *testing.T) *QuicConnection {
	t.Helper()
	ctrl := gomock.NewController(t)
	session := mocks.NewMockSession(ctrl)
	session.EXPECT().Close().Return(nil).AnyTimes()
	return NewQuicConn(&ConnectionOptions{
		ConnType: api.ProtocolTypeQuic,
		ConnUse:  api.UseTypeMessage,
		Base:     session,
		CtrlLane: lane.NewLane(api.ProtocolTypeQuic, mocks.NewMockStream(ctrl)),
		State:    &ConnectionState{State: api.StatConnected},
	})
}

// TestNewQuicConnWiresMessageFifo guards the wiring Close depends on: without
// a fifo the connection would panic on teardown instead of releasing readers.
func TestNewQuicConnWiresMessageFifo(t *testing.T) {
	conn := newTestQuicConn(t)

	if conn.messageFifo == nil {
		t.Fatal("NewQuicConn left messageFifo nil")
	}
	if conn.syncKeeper == nil {
		t.Fatal("NewQuicConn left syncKeeper nil")
	}
	if conn.streamManager == nil {
		t.Fatal("NewQuicConn left streamManager nil")
	}
}

// TestCloseReleasesBlockedReadMessage guards the invariant that tearing the
// connection down releases its reader. ReadMessage parks on the message fifo,
// and EdgeHub's reconnect loop calls UnInit (which reaches Close) before
// starting the next connection: a reader that is never released leaks one
// goroutine per reconnect and keeps holding the previous connection.
func TestCloseReleasesBlockedReadMessage(t *testing.T) {
	conn := newTestQuicConn(t)

	blocked := make(chan error, 1)
	go func() {
		msg := &model.Message{}
		blocked <- conn.ReadMessage(msg)
	}()

	// Confirm the reader is parked, so the test cannot pass with a Close
	// that only makes later reads fail.
	select {
	case err := <-blocked:
		t.Fatalf("ReadMessage returned before Close: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case err := <-blocked:
		if err == nil {
			t.Fatal("ReadMessage returned a nil error after Close, want the fifo error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ReadMessage is still blocked after Close; the reader was not released")
	}
}

// TestCloseIsIdempotent covers the teardown being reached twice on the server
// side, where serveSession closes the connection when AcceptStream fails and
// the owner closes it again afterwards.
func TestCloseIsIdempotent(t *testing.T) {
	conn := newTestQuicConn(t)

	if err := conn.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if conn.state.State != api.StatDisconnected {
		t.Fatalf("state = %q, want %q", conn.state.State, api.StatDisconnected)
	}
}
