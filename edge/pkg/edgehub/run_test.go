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

package edgehub

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"k8s.io/client-go/util/flowcontrol"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

// runHarness wires an EdgeHub to a mock cloud client and records the
// connection lifecycle so the reconnect loop can be driven deterministically.
// By default Receive and Send park until release is closed, so the routing
// goroutines neither error out nor trigger a reconnect on their own; a test
// overrides sendFn / recvFn before start to inject transport failures.
type runHarness struct {
	eh        *EdgeHub
	mock      *edgehub.MockAdapter
	inits     atomic.Int32
	uninits   atomic.Int32
	connected chan struct{} // one token per successful Init
	uninited  chan struct{} // one token per UnInit
	release   chan struct{} // closed at cleanup; unblocks the parked Receive/Send calls
	sendFn    func(model.Message) error
	recvFn    func() (model.Message, error)
}

func newRunHarness(t *testing.T) *runHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &runHarness{
		mock:      edgehub.NewMockAdapter(ctrl),
		connected: make(chan struct{}, 16),
		uninited:  make(chan struct{}, 16),
		release:   make(chan struct{}),
	}
	h.sendFn = func(model.Message) error {
		<-h.release
		return errors.New("released")
	}
	h.recvFn = func() (model.Message, error) {
		<-h.release
		return model.Message{}, errors.New("released")
	}
	// routeToCloud blocks in beehiveContext.Receive, which cannot be
	// interrupted and would otherwise outlive the test and take a message
	// meant for a later one. Park it on release like the mock transport.
	patches := gomonkey.ApplyFunc(beehiveContext.Receive, func(string) (model.Message, error) {
		<-h.release
		return model.Message{}, errors.New("released")
	})
	t.Cleanup(patches.Reset)
	// Runs before the controller's Finish (cleanups are LIFO), so the routing
	// goroutines return and exit before the mock is verified.
	t.Cleanup(func() { close(h.release) })

	// No test lets a live connection's Send succeed, so keepalive never
	// reaches its heartbeat wait; the long period is insurance against a
	// heartbeat ever driving the loop instead of the harness.
	heartbeat := config.Config.Heartbeat
	config.Config.Heartbeat = 3600
	t.Cleanup(func() { config.Config.Heartbeat = heartbeat })

	h.eh = &EdgeHub{
		reconnectChan: make(chan struct{}, 1),
		rotateChan:    make(chan struct{}, 1),
		rateLimiter:   flowcontrol.NewTokenBucketRateLimiter(10, 10),
		newClient:     func() (clients.Adapter, error) { return h.mock, nil },
	}
	h.mock.EXPECT().UnInit().Do(func() {
		h.uninits.Add(1)
		h.uninited <- struct{}{}
	}).AnyTimes()
	h.mock.EXPECT().Send(gomock.Any()).DoAndReturn(func(msg model.Message) error {
		return h.sendFn(msg)
	}).AnyTimes()
	h.mock.EXPECT().Receive().DoAndReturn(func() (model.Message, error) {
		return h.recvFn()
	}).AnyTimes()
	return h
}

// expectConnect makes Init succeed and records each connection.
func (h *runHarness) expectConnect() {
	h.mock.EXPECT().Init().DoAndReturn(func() error {
		h.inits.Add(1)
		h.connected <- struct{}{}
		return nil
	}).AnyTimes()
}

// start runs the reconnect loop until done is closed; exited is closed when
// run returns.
func (h *runHarness) start(done <-chan struct{}) (exited chan struct{}) {
	exited = make(chan struct{})
	go func() {
		h.eh.run(done)
		close(exited)
	}()
	return exited
}

func awaitSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// TestRunReturnsWhenDoneAlreadyClosed verifies that the loop honours a
// shutdown requested before the first connection attempt: no client is built.
func TestRunReturnsWhenDoneAlreadyClosed(t *testing.T) {
	h := newRunHarness(t) // no Init expectation: an attempt would fail the mock
	done := make(chan struct{})
	close(done)
	awaitSignal(t, h.start(done), "run to return")
	if h.inits.Load() != 0 {
		t.Fatalf("Init called %d times after shutdown was requested", h.inits.Load())
	}
}

// TestRunRetriesInitAfterBackoff verifies the connect-failure path: a failing
// Init is retried after the initial backoff interval, and closing done during
// a backoff wait returns promptly instead of sleeping out the interval.
func TestRunRetriesInitAfterBackoff(t *testing.T) {
	h := newRunHarness(t)
	initFailed := make(chan time.Time, 16)
	h.mock.EXPECT().Init().DoAndReturn(func() error {
		h.inits.Add(1)
		initFailed <- time.Now()
		return errors.New("connection refused")
	}).AnyTimes()

	done := make(chan struct{})
	exited := h.start(done)
	var first, second time.Time
	select {
	case first = <-initFailed:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the first failed Init")
	}
	select {
	case second = <-initFailed:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the retry after the backoff")
	}
	// reconnectBackoff starts at 2s (+ up to 20% jitter); anything shorter
	// means the failure path skipped the backoff wait.
	if gap := second.Sub(first); gap < 1500*time.Millisecond {
		t.Fatalf("retry came after %v, want at least the initial backoff", gap)
	}

	started := time.Now()
	close(done)
	awaitSignal(t, exited, "run to return during the backoff wait")
	if time.Since(started) > time.Second {
		t.Fatalf("run took %v to stop; it must not sleep out the backoff after done is closed", time.Since(started))
	}
	if h.uninits.Load() != 0 {
		t.Fatalf("UnInit called %d times without a successful connect", h.uninits.Load())
	}
}

// TestRunReconnectsOnTransportError verifies the transport-failure path: a
// keepalive write error triggers UnInit, the connection is re-established
// after the backoff interval, and closing done during a backoff wait returns
// promptly.
func TestRunReconnectsOnTransportError(t *testing.T) {
	h := newRunHarness(t)
	h.expectConnect()
	// keepalive's first Send on every connection fails and requests a reconnect.
	h.sendFn = func(model.Message) error { return errors.New("write: broken pipe") }

	done := make(chan struct{})
	exited := h.start(done)
	awaitSignal(t, h.connected, "the first connect")
	awaitSignal(t, h.uninited, "UnInit after the transport error")
	uninitedAt := time.Now()
	awaitSignal(t, h.connected, "the reconnect after the backoff")
	// reconnectBackoff starts at 2s (+ up to 20% jitter); anything shorter
	// means the transport-error path skipped the backoff wait.
	if gap := time.Since(uninitedAt); gap < 1500*time.Millisecond {
		t.Fatalf("reconnected after %v, want at least the initial backoff", gap)
	}
	awaitSignal(t, h.uninited, "UnInit after the second transport error")

	started := time.Now()
	close(done)
	awaitSignal(t, exited, "run to return during the reconnect backoff")
	if time.Since(started) > time.Second {
		t.Fatalf("run took %v to stop; it must not sleep out the backoff after done is closed", time.Since(started))
	}
	if got := h.inits.Load(); got != 2 {
		t.Fatalf("Init called %d times, want 2 (initial + one reconnect)", got)
	}
}

// TestRunReconnectsImmediatelyOnRotation verifies that a certificate rotation
// tears the connection down and reconnects without a backoff wait, and that
// shutting down while connected cleans up (UnInit) before returning.
func TestRunReconnectsImmediatelyOnRotation(t *testing.T) {
	h := newRunHarness(t)
	h.expectConnect()

	done := make(chan struct{})
	exited := h.start(done)
	awaitSignal(t, h.connected, "the first connect")

	h.eh.rotateChan <- struct{}{}
	awaitSignal(t, h.uninited, "UnInit after the rotation signal")
	rotatedAt := time.Now()
	awaitSignal(t, h.connected, "the reconnect after rotation")
	if d := time.Since(rotatedAt); d > time.Second {
		t.Fatalf("reconnect after rotation took %v; it must not wait a backoff interval", d)
	}

	close(done)
	awaitSignal(t, h.uninited, "UnInit on shutdown while connected")
	awaitSignal(t, exited, "run to return after done")
	if got := h.inits.Load(); got != 2 {
		t.Fatalf("Init called %d times, want 2 (initial + post-rotation)", got)
	}
}

// TestRunDropsStaleRotationBeforeConnect verifies that a rotation which
// completed while disconnected does not cause a redundant reconnect (the
// upcoming Init already loads the newest certificate), and that shutdown from
// the connected state performs the cleanup exactly once.
func TestRunDropsStaleRotationBeforeConnect(t *testing.T) {
	h := newRunHarness(t)
	h.expectConnect()
	h.eh.rotateChan <- struct{}{} // stale: arrived before the connect

	done := make(chan struct{})
	exited := h.start(done)
	awaitSignal(t, h.connected, "the connect")

	select {
	case <-h.uninited:
		t.Fatal("a rotation that completed before the connect must not trigger a reconnect")
	case <-time.After(200 * time.Millisecond):
	}

	close(done)
	awaitSignal(t, exited, "run to return after done")
	if got := h.uninits.Load(); got != 1 {
		t.Fatalf("UnInit called %d times, want exactly 1 (shutdown cleanup)", got)
	}
	if got := h.inits.Load(); got != 1 {
		t.Fatalf("Init called %d times, want 1", got)
	}
}
