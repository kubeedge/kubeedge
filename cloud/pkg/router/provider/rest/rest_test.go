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

package rest

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockingTarget simulates a target (like the rest and eventbus targets) that
// ignores the stop channel and stays busy until it is released.
type blockingTarget struct {
	entered chan struct{}
	release chan struct{}
}

func (b *blockingTarget) Name() string { return "blocking-target" }

func (b *blockingTarget) GoToTarget(_ map[string]interface{}, _ chan struct{}) (interface{}, error) {
	close(b.entered)
	<-b.release
	return nil, nil
}

// TestForwardTimeoutDoesNotHangOrLeak verifies that when the target is slow and
// ignores the stop channel, Forward still returns on timeout (it does not block
// forever on the stop send), and the worker goroutine exits instead of blocking
// forever on the result channel.
func TestForwardTimeoutDoesNotHangOrLeak(t *testing.T) {
	r := &Rest{Path: "rest"}
	target := &blockingTarget{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/default/node1/rest/foo", nil)
	data := map[string]interface{}{
		"request":   req,
		"timeout":   50 * time.Millisecond,
		"data":      []byte("payload"),
		"messageID": "msg-1",
	}

	before := runtime.NumGoroutine()

	// Forward must return on timeout even though the target ignores stop; before
	// the fix, the stop send blocked here forever.
	resp, err := r.Forward(target, data)
	require.NoError(t, err)
	httpResp, ok := resp.(*http.Response)
	require.True(t, ok)
	assert.Equal(t, http.StatusRequestTimeout, httpResp.StatusCode)

	// The worker goroutine is still running the target; release it and confirm it
	// exits instead of blocking forever on the result channel. Poll from this
	// goroutine so the check itself does not inflate the goroutine count.
	<-target.entered
	close(target.release)

	after := before
	leaked := true
	for i := 0; i < 200; i++ {
		after = runtime.NumGoroutine()
		if after <= before {
			leaked = false
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Falsef(t, leaked, "worker goroutine leaked after timeout: before=%d after=%d", before, after)
}
