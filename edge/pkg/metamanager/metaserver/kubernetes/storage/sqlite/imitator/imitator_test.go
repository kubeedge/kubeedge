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

package imitator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/watchhook"
)

// TestWatchReturnsClosedChannelWhenHookRegistrationFails verifies that a
// failed hook registration yields a closed, non-nil channel, so a caller
// ranging over it returns immediately instead of blocking forever.
func TestWatchReturnsClosedChannelWhenHookRegistrationFails(t *testing.T) {
	orig := newWatchHook
	defer func() { newWatchHook = orig }()
	newWatchHook = func(string, uint64, watchhook.Receiver) (*watchhook.WatchHook, error) {
		return nil, errors.New("hook already registered")
	}

	wch := (&imitator{}).Watch(context.Background(), "/core/v1/pods/default/test", 0)
	if wch == nil {
		t.Fatal("expected a non-nil channel when hook registration fails")
	}

	select {
	case e, ok := <-wch:
		if ok {
			t.Fatalf("expected the channel to be closed, got event %v", e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("receiving from the returned channel blocked")
	}
}

// TestWatchClosesChannelOnContextCancel verifies the regular path is unchanged:
// the channel stays open while watching and is closed once ctx is cancelled.
func TestWatchClosesChannelOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wch := (&imitator{}).Watch(ctx, "/core/v1/pods/default/test", 0)
	if wch == nil {
		t.Fatal("expected a non-nil channel")
	}

	select {
	case e := <-wch:
		t.Fatalf("expected no event before cancel, got %v", e)
	case <-time.After(100 * time.Millisecond):
	}

	cancel()
	select {
	case _, ok := <-wch:
		if ok {
			t.Fatal("expected the channel to be closed after cancel")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("channel was not closed after cancel")
	}
}
