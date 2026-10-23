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

// TestWatchClosedSourceTerminatesCleanly verifies that when the imitator
// returns an already closed channel, as it does when hook registration fails,
// the watch terminates within a bounded time with a clean ResultChan close and
// no watch.Error event. Previously both an error and a normal close were
// signalled, so callers saw either outcome at random; the loop makes sure the
// result is deterministic.
func TestWatchClosedSourceTerminatesCleanly(t *testing.T) {
	client := fake.Client{
		ListF: func(context.Context, string) (imitator.Resp, error) {
			return imitator.Resp{Kvs: &[]models.MetaV2{}}, nil
		},
		WatchF: func(context.Context, string, uint64) <-chan watch.Event {
			wch := make(chan watch.Event)
			close(wch)
			return wch
		},
	}
	w := newWatcher(client, nil)

	for i := 0; i < 100; i++ {
		wc := w.createWatchChan(context.Background(), "/core/v1/pods/default", 0, true, storage.Everything)
		go wc.run()

		select {
		case e, ok := <-wc.ResultChan():
			if ok {
				t.Fatalf("iteration %d: expected a clean close, got event %s", i, e.Type)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("iteration %d: watch did not terminate", i)
		}

		// run closes ResultChan only after cancelling wc.ctx and waiting for
		// processEvent, and startWatching has already returned, so no
		// goroutine of this watch is left running.
		if wc.ctx.Err() == nil {
			t.Fatalf("iteration %d: expected the watch context to be cancelled", i)
		}
	}
}
