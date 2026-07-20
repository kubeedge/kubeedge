package imitator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/watchhook"
)

func TestWatchReturnsClosedChannelOnRegistrationFailure(t *testing.T) {
	original := registerWatchHook
	registerWatchHook = func(key string, rev uint64, receiver watchhook.Receiver) (*watchhook.WatchHook, error) {
		return nil, errors.New("hook registration failed")
	}
	defer func() { registerWatchHook = original }()

	s := &imitator{}
	ch := s.Watch(context.TODO(), "/core/v1/pods/default/foo", 1)
	if ch == nil {
		t.Fatal("Watch returned a nil channel on registration failure")
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected the returned channel to be closed without events")
		}
	case <-time.After(time.Second):
		t.Fatal("receive from the returned channel did not complete immediately")
	}
}
