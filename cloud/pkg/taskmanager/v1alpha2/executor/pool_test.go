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

package executor

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	var count atomic.Int32
	var finish bool
	p := NewPool(2)
	go func() {
		for {
			if finish {
				break
			}
			if count.Load() == 2 {
				count.Store(0)
				p.Release()
				p.Release()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	for i := 0; i < 10; i++ {
		p.Acquire()
		t.Logf("acquire %d", i)
		count.Add(1)
	}
	finish = true
	time.Sleep(1 * time.Second)
}
