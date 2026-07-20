package watchhook

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

type testReceiver struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *testReceiver) Receive(watch.Event) error {
	r.calls.Add(1)
	if r.started != nil {
		r.once.Do(func() {
			close(r.started)
		})
	}
	if r.release != nil {
		<-r.release
	}
	return nil
}

func newTestEvent(t *testing.T) (watch.Event, string) {
	t.Helper()

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("Pod")
	obj.SetNamespace("default")
	obj.SetName("test-pod")
	obj.SetResourceVersion("1")

	key, err := metaserver.KeyFuncObj(obj)
	require.NoError(t, err)
	return watch.Event{Type: watch.Modified, Object: obj}, key
}

func TestWatchHookNoDeliveryAfterStop(t *testing.T) {
	event, key := newTestEvent(t)
	receiver := &testReceiver{}
	hook, err := NewWatchHook(key, 0, receiver)
	require.NoError(t, err)

	hook.Stop()
	require.NoError(t, hook.Do(event))
	Trigger(event)
	hook.Stop()

	require.Zero(t, receiver.calls.Load())
}

func TestWatchHookStopWaitsForInFlightDelivery(t *testing.T) {
	event, key := newTestEvent(t)
	receiver := &testReceiver{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	hook, err := NewWatchHook(key, 0, receiver)
	require.NoError(t, err)

	doDone := make(chan struct{})
	doErr := make(chan error, 1)
	go func() {
		defer close(doDone)
		doErr <- hook.Do(event)
	}()
	<-receiver.started

	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		hook.Stop()
	}()

	select {
	case <-stopDone:
		t.Fatal("Stop returned before the in-flight delivery completed")
	case <-time.After(100 * time.Millisecond):
	}

	close(receiver.release)
	<-doDone
	require.NoError(t, <-doErr)
	<-stopDone

	require.NoError(t, hook.Do(event))
	require.Equal(t, int32(1), receiver.calls.Load())
}

func TestWatchHookConcurrentTriggerAndStop(t *testing.T) {
	event, key := newTestEvent(t)
	const (
		workers    = 8
		iterations = 100
	)

	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers + 1)

	go func() {
		defer wg.Done()
		<-start
		for range workers * iterations {
			Trigger(event)
		}
	}()

	for range workers {
		go func() {
			defer wg.Done()
			<-start
			for range iterations {
				hook, err := NewWatchHook(key, 0, &testReceiver{})
				if err != nil {
					errs <- err
					return
				}
				hook.Stop()
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
