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

package edgestream

import (
	"sync"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

// The tunnel session's reader goroutine delivers cloud messages to a local connection while the goroutine that served
// that connection removes it once the exec/attach/logs/metrics session ends. Delivery must neither send on the
// closed read channel ("send on closed channel" crashes edgecore) nor block the removal when the read channel is full.
func TestWriteToLocalConnectionDuringConnectionRemoval(t *testing.T) {
	connections := map[string]func() stream.EdgedConnection{
		"exec": func() stream.EdgedConnection {
			return &stream.EdgedExecConnection{ReadChan: make(chan *stream.Message, 1), Stop: make(chan struct{}, 2)}
		},
		"attach": func() stream.EdgedConnection {
			return &stream.EdgedAttachConnection{ReadChan: make(chan *stream.Message, 1), Stop: make(chan struct{}, 2)}
		},
		"logs": func() stream.EdgedConnection {
			return &stream.EdgedLogsConnection{ReadChan: make(chan *stream.Message, 1), Stop: make(chan struct{}, 2)}
		},
		"metrics": func() stream.EdgedConnection {
			return &stream.EdgedMetricsConnection{ReadChan: make(chan *stream.Message, 1), Stop: make(chan struct{}, 2)}
		},
	}
	for name, newConnection := range connections {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 500; i++ {
				session := &TunnelSession{localCons: map[uint64]stream.EdgedConnection{}}
				session.AddLocalConnection(1, newConnection())
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					// Nobody reads the channel (the connection's reader already stopped): the second message fills
					// a full one-slot buffer, so the sender is waiting when the connection is removed.
					for j := 0; j < 3; j++ {
						session.WriteToLocalConnection(stream.NewMessage(1, stream.MessageTypeData, []byte("x")))
					}
				}()
				go func() {
					defer wg.Done()
					session.DeleteLocalConnection(1)
				}()
				done := make(chan struct{})
				go func() {
					wg.Wait()
					close(done)
				}()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Fatalf("iteration %d: delivery and connection removal blocked each other", i)
				}
			}
		})
	}
}
