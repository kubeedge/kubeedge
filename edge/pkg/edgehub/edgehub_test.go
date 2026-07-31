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

package edgehub

import (
	"testing"
	"time"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

func TestGetCertSyncChannel(t *testing.T) {
	t.Run("GetCertSyncChannel()", func(t *testing.T) {
		certSync := GetCertSyncChannel()
		if certSync != nil {
			t.Errorf("GetCertSyncChannel() returned unexpected result. got = %v, want = %v", certSync, nil)
		}
	})
}

func TestNewCertSyncChannel(t *testing.T) {
	t.Run("NewCertSyncChannel()", func(t *testing.T) {
		certSync := NewCertSyncChannel()
		if len(certSync) != 1 {
			t.Errorf("NewCertSyncChannel() returned  unexpected results. size got = %d, size want = 2", len(certSync))
		}
		if _, ok := certSync["edgestream"]; !ok {
			t.Error("NewCertSyncChannel() returned  unexpected results. expected key edgestream to be present but it was not available.")
		}
	})
}

func TestRegister(t *testing.T) {
	tests := []struct {
		eh           *v1alpha2.EdgeHub
		nodeName     string
		name         string
		wantNodeName string
	}{
		{
			name:         "",
			nodeName:     "test1",
			wantNodeName: "test1",
			eh: &v1alpha2.EdgeHub{
				WebSocket: &v1alpha2.EdgeHubWebSocket{
					Server: "localhost:8080",
				},
				ProjectID: "test_id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Register(tt.eh, tt.nodeName)

			if config.Config.NodeName != tt.wantNodeName {
				t.Errorf("failed to Register(). Nodename : got = %s, want = %s", config.Config.NodeName, tt.wantNodeName)
			}
		})
	}
}

func TestName(t *testing.T) {
	t.Run("EdgeHub.Name()", func(t *testing.T) {
		if got := (&EdgeHub{}).Name(); got != "websocket" {
			t.Errorf("EdgeHub.Name() returned unexpected result. got = %s, want = websocket", got)
		}
	})
}

func TestGroup(t *testing.T) {
	t.Run("EdgeHub.Group()", func(t *testing.T) {
		if got := (&EdgeHub{}).Group(); got != "hub" {
			t.Errorf("EdgeHub.Group() returned unexpected result. got = %s, want = hub", got)
		}
	})
}

func TestTriggerReconnectNonBlocking(t *testing.T) {
	eh := &EdgeHub{reconnectChan: make(chan struct{}, 1)}

	// First call enqueues a signal.
	eh.triggerReconnect()
	// Second call must not block even though the channel is full.
	done := make(chan struct{})
	go func() {
		eh.triggerReconnect()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("triggerReconnect blocked when channel was full")
	}

	// Exactly one signal should be receivable; the second send was coalesced.
	select {
	case <-eh.reconnectChan:
	default:
		t.Fatal("expected one reconnect signal")
	}
	select {
	case <-eh.reconnectChan:
		t.Fatal("did not expect a second reconnect signal")
	default:
	}
}

// TestIfRotationDoneSendsToRotateChan guards the wiring that makes the
// rotation guarantee hold end-to-end: a certManager.Done signal must be
// forwarded to rotateChan — never to the drainable reconnectChan, where the
// post-connect drain could discard it.
func TestIfRotationDoneSendsToRotateChan(t *testing.T) {
	eh := &EdgeHub{
		reconnectChan: make(chan struct{}, 1),
		rotateChan:    make(chan struct{}, 1),
		certManager: certificate.CertManager{
			RotateCertificates: true,
			Done:               make(chan struct{}),
		},
	}
	go eh.ifRotationDone()

	select {
	case eh.certManager.Done <- struct{}{}:
	case <-time.After(time.Second):
		t.Fatal("ifRotationDone did not consume certManager.Done")
	}

	select {
	case <-eh.rotateChan:
	case <-time.After(time.Second):
		t.Fatal("rotation signal was not forwarded to rotateChan")
	}
	select {
	case <-eh.reconnectChan:
		t.Fatal("rotation signal must not go to reconnectChan")
	default:
	}
}

// TestDrainReconnectLeavesRotateSignal guards the certificate-rotation
// guarantee: the post-connect drain must only discard stale transport
// reconnect signals, never a pending rotation signal. Dropping the latter
// would leave a connection running on a stale certificate until the next
// natural disconnect (certManager.Done fires only once per rotation).
func TestDrainReconnectLeavesRotateSignal(t *testing.T) {
	eh := &EdgeHub{
		reconnectChan: make(chan struct{}, 1),
		rotateChan:    make(chan struct{}, 1),
	}
	eh.triggerReconnect()
	eh.rotateChan <- struct{}{}

	eh.drainReconnect()

	select {
	case <-eh.reconnectChan:
		t.Fatal("drainReconnect must discard the pending reconnect signal")
	default:
	}
	select {
	case <-eh.rotateChan:
	default:
		t.Fatal("drainReconnect must not consume a pending rotation signal")
	}

	// Draining with nothing pending must not block.
	done := make(chan struct{})
	go func() {
		eh.drainReconnect()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("drainReconnect blocked on an empty channel")
	}
}

// TestShouldResetBackoff guards the flapping protection: the backoff is only
// reset once a connection survived longer than the backoff cap, so a
// connect-then-die loop keeps backing off instead of retrying at the initial
// interval forever.
func TestShouldResetBackoff(t *testing.T) {
	b := reconnectBackoff()
	if shouldResetBackoff(time.Now().Add(-time.Second), b) {
		t.Fatal("a connection that lived 1s must not reset the backoff")
	}
	if !shouldResetBackoff(time.Now().Add(-b.Cap-time.Second), b) {
		t.Fatal("a connection that outlived the cap must reset the backoff")
	}
}

func TestReconnectBackoffGrowsAndCaps(t *testing.T) {
	b := reconnectBackoff()
	// Derive bounds from the Backoff contract itself so the test keeps
	// holding when the constants are tuned.
	maxAllowed := b.Cap + time.Duration(float64(b.Cap)*b.Jitter)

	for i := 0; i < 50; i++ {
		got := b.Step()
		if got <= 0 {
			t.Fatalf("step %d: backoff must be positive, got %v", i, got)
		}
		if got > maxAllowed {
			t.Fatalf("step %d: %v exceeds cap+jitter (%v)", i, got, maxAllowed)
		}
	}
}

func TestReconnectBackoffResetReturnsInitial(t *testing.T) {
	b := reconnectBackoff()
	// Exhaust until Cap is reached.
	for i := 0; i < 10; i++ {
		b.Step()
	}
	// Re-creating returns a backoff starting from the initial duration;
	// this is the contract used to reset after a healthy connection.
	// Bounds derive from the Backoff fields (captured before Step mutates
	// Duration) so the test keeps holding when the constants are tuned.
	b2 := reconnectBackoff()
	minAllowed := b2.Duration
	maxAllowed := b2.Duration + time.Duration(float64(b2.Duration)*b2.Jitter)
	first := b2.Step()
	if first < minAllowed || first > maxAllowed {
		t.Fatalf("initial step out of [%v, %v]: got %v", minAllowed, maxAllowed, first)
	}
}

func TestEnable(t *testing.T) {
	tests := []struct {
		eh   *EdgeHub
		want bool
		name string
	}{
		{
			name: "Enable true",
			want: true,
			eh:   &EdgeHub{enable: true},
		},
		{
			name: "Enable false",
			want: false,
			eh:   &EdgeHub{enable: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.eh.Enable(); got != tt.want {
				t.Errorf("EdgeHub.Enable() returned expected results. got = %v, want = %v", got, tt.want)
			}
		})
	}
}
