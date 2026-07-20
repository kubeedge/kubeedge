package sqlite

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/fake"
)

// TestWatchClosedSourceChannelTerminatesCleanly verifies that when the
// underlying client returns a pre-closed channel, as imitator.Watch does when
// hook registration fails, the watch terminates within a bounded time with a
// clean ResultChan close and no watch.Error event.
func TestWatchClosedSourceChannelTerminatesCleanly(t *testing.T) {
	closedCh := make(chan watch.Event)
	close(closedCh)
	kvs := []models.MetaV2{}
	client := fake.Client{
		ListF: func(ctx context.Context, key string) (imitator.Resp, error) {
			return imitator.Resp{Kvs: &kvs, Revision: 0}, nil
		},
		WatchF: func(ctx context.Context, key string, rv uint64) <-chan watch.Event {
			return closedCh
		},
	}

	w := newWatcher(client, nil)
	wi, err := w.Watch(context.TODO(), "/core/v1/pods/default", 0, true, storage.Everything)
	if err != nil {
		t.Fatalf("Watch returned error: %v", err)
	}

	timeout := time.After(3 * time.Second)
	for {
		select {
		case e, ok := <-wi.ResultChan():
			if !ok {
				return
			}
			if e.Type == watch.Error {
				t.Fatalf("expected clean close, got error event: %v", e)
			}
			t.Fatalf("expected no events, got: %v", e)
		case <-timeout:
			t.Fatal("watch did not terminate within bounded time")
		}
	}
}
