package watchhook

import (
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type recorder struct {
	events []watch.Event
}

func (r *recorder) Receive(e watch.Event) error {
	r.events = append(r.events, e)
	return nil
}

func newPod(name, rv string) *unstructured.Unstructured {
	pod := &unstructured.Unstructured{}
	pod.SetAPIVersion("v1")
	pod.SetKind("Pod")
	pod.SetNamespace("default")
	pod.SetName(name)
	pod.SetResourceVersion(rv)
	return pod
}

func TestTriggerResourceVersion(t *testing.T) {
	tests := []struct {
		name      string
		eventType watch.EventType
		rv        string
		checkRV   bool
		delivered bool
	}{
		{"modified above the watch revision", watch.Modified, "150", true, true},
		{"modified below the watch revision", watch.Modified, "50", true, false},
		{"deleted above the watch revision", watch.Deleted, "150", true, true},
		{"deleted below the watch revision", watch.Deleted, "50", true, false},
		{"deleted without resourceVersion (from synccontroller.gcOrphanedObjectSync, late)", watch.Deleted, "", true, false},
		{"deleted with the stored object below the watch revision", watch.Deleted, "50", false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := &recorder{}
			hook, err := NewWatchHook("/core/v1/pods/default", 100, r)
			if err != nil {
				t.Fatal(err)
			}
			defer hook.Stop()

			Trigger(watch.Event{Type: test.eventType, Object: newPod("p", test.rv)}, test.checkRV)

			if got := len(r.events) == 1; got != test.delivered {
				t.Errorf("delivered = %v, want %v", got, test.delivered)
			}
		})
	}
}

// Watches open and close, as imitator.Watch does, while events are triggered: Trigger
// must not read hooks while they are written, nor send on a closed channel.
func TestTriggerWhileWatchesStop(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20000; i++ {
			Trigger(watch.Event{Type: watch.Modified, Object: newPod("p", "")}, true)
		}
	}()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				ch := make(chan watch.Event)
				hook, err := NewWatchHook("/core/v1/pods/default", 0, NewChanReceiver(ch))
				if err != nil {
					t.Error(err)
					return
				}
				go func() {
					for range ch {
					}
				}()
				hook.Stop()
				close(ch)
			}
		}()
	}
	wg.Wait()
}
