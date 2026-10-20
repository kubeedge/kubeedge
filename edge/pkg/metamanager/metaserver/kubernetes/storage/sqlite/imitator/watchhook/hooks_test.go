package watchhook

import (
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

func newPod(name, rv string) *unstructured.Unstructured {
	pod := &unstructured.Unstructured{}
	pod.SetAPIVersion("v1")
	pod.SetKind("Pod")
	pod.SetNamespace("default")
	pod.SetName(name)
	pod.SetResourceVersion(rv)
	return pod
}

// Watches open and close, as imitator.Watch does, while events are triggered: Trigger
// must not read hooks while they are written, nor send on a closed channel.
func TestTriggerWhileWatchesStop(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20000; i++ {
			Trigger(watch.Event{Type: watch.Modified, Object: newPod("p", "")})
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
