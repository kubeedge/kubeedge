package sqlite

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	fakeclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/fake"
)

func podJSON(name string) string {
	return fmt.Sprintf(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":%q,"namespace":"default"}}`, name)
}

// neverWatch returns a watch channel that stays open until ctx is cancelled.
func neverWatch(ctx context.Context) <-chan watch.Event {
	ch := make(chan watch.Event)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch
}

func makeTestWatcher(kvs []models.MetaV2, revision uint64) (*watcher, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	resp := imitator.Resp{
		Kvs:      &kvs,
		Revision: revision,
	}
	fc := fakeclient.Client{
		ListF: func(_ context.Context, _ string) (imitator.Resp, error) {
			return resp, nil
		},
		WatchF: func(c context.Context, _ string, _ uint64) <-chan watch.Event {
			return neverWatch(c)
		},
	}
	_ = ctx
	return newWatcher(fc, unstructured.UnstructuredJSONScheme), cancel
}

// collectEvents reads up to maxEvents from ch or until timeout, stopping after the first BOOKMARK.
func collectEvents(ch <-chan watch.Event, maxEvents int, timeout time.Duration) []watch.Event {
	var events []watch.Event
	deadline := time.After(timeout)
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, e)
			if e.Type == watch.Bookmark || len(events) >= maxEvents {
				return events
			}
		case <-deadline:
			return events
		}
	}
}

// TestWatchBookmarkAfterInitialSync verifies that after sync(), all Added events arrive
// before the synthesised BOOKMARK, and that the BOOKMARK carries the correct ResourceVersion
// and the k8s.io/initial-events-end annotation required by client-go >= 1.30.
func TestWatchBookmarkAfterInitialSync(t *testing.T) {
	kvs := []models.MetaV2{
		{Value: podJSON("pod-1")},
		{Value: podJSON("pod-2")},
	}
	const rev uint64 = 42

	w, cancel := makeTestWatcher(kvs, rev)
	defer cancel()

	wi, err := w.Watch(context.Background(), "/core/v1/pods/default/", 0, true, storage.Everything)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}
	defer wi.Stop()

	events := collectEvents(wi.ResultChan(), 10, 3*time.Second)

	var addedCount int
	var bookmarkIdx = -1
	for i, e := range events {
		switch e.Type {
		case watch.Added:
			if bookmarkIdx != -1 {
				t.Errorf("Added event at index %d appeared after BOOKMARK at index %d", i, bookmarkIdx)
			}
			addedCount++
		case watch.Bookmark:
			bookmarkIdx = i
		}
	}

	if addedCount != len(kvs) {
		t.Errorf("want %d Added events, got %d", len(kvs), addedCount)
	}
	if bookmarkIdx == -1 {
		t.Fatal("no BOOKMARK event received")
	}

	bm := events[bookmarkIdx]
	accessor, ok := bm.Object.(*metav1.PartialObjectMetadata)
	if !ok {
		t.Fatalf("BOOKMARK object is %T, want *metav1.PartialObjectMetadata", bm.Object)
	}
	if accessor.ResourceVersion != fmt.Sprintf("%d", rev) {
		t.Errorf("BOOKMARK ResourceVersion = %q, want %q", accessor.ResourceVersion, fmt.Sprintf("%d", rev))
	}
	if accessor.Annotations["k8s.io/initial-events-end"] != "true" {
		t.Errorf("BOOKMARK missing k8s.io/initial-events-end annotation, got %v", accessor.Annotations)
	}
}

// TestProcessEventBookmarkBypass verifies that processEvent forwards watch.Bookmark events
// directly to resultChan without attempting KeyFuncObj (which would error and drop them).
func TestProcessEventBookmarkBypass(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wc := &watchChan{
		watcher:           &watcher{},
		internalPred:      storage.Everything,
		incomingEventChan: make(chan *watch.Event, incomingBufSize),
		resultChan:        make(chan watch.Event, outgoingBufSize),
		errChan:           make(chan error, 1),
		added:             make(map[string]bool),
		ctx:               ctx,
		cancel:            cancel,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go wc.processEvent(&wg)

	bm := &watch.Event{
		Type: watch.Bookmark,
		Object: &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{
				ResourceVersion: "99",
				Annotations:     map[string]string{"k8s.io/initial-events-end": "true"},
			},
		},
	}
	wc.incomingEventChan <- bm

	select {
	case got := <-wc.resultChan:
		if got.Type != watch.Bookmark {
			t.Errorf("resultChan event type = %v, want Bookmark", got.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for BOOKMARK in resultChan")
	}
}
