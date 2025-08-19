/*
Copyright 2025 The KubeEdge Authors.

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
package priority

import (
	"sync"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestPriorityQueue_OrderAndFIFO(t *testing.T) {
	q := NewMessagePriorityQueue()
	mk := func(p int32, id string) model.Message {
		m := model.NewMessage("")
		m.SetPriority(p)
		m.BuildRouter("src", "grp", id, "op")
		return *m
	}
	// same priority to test FIFO
	q.Add(mk(model.PriorityImportant, "a1"))
	q.Add(mk(model.PriorityImportant, "a2"))
	// mixed priorities
	q.Add(mk(model.PriorityLow, "low"))
	q.Add(mk(model.PriorityUrgent, "urgent"))
	q.Add(mk(model.PriorityNormal, "normal"))

	expects := []string{"urgent", "a1", "a2", "normal", "low"}
	for i := range expects {
		msg, ok := q.Get()
		if !ok {
			t.Fatalf("expected ok on get at %d", i)
		}
		if got := msg.GetResource(); got != expects[i] {
			t.Fatalf("order mismatch at %d: got %s expect %s", i, got, expects[i])
		}
	}
}

func TestPriorityQueue_CloseUnblocksWaiters(t *testing.T) {
	q := NewMessagePriorityQueue()
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		_, ok := q.Get()
		if ok {
			t.Errorf("expected ok=false after Close")
		}
		close(done)
	}()
	// ensure goroutine is waiting
	time.Sleep(20 * time.Millisecond)
	q.Close()
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Get waiter was not unblocked by Close")
	}
	wg.Wait()
}
