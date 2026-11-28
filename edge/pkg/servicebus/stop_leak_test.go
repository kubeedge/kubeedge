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

package servicebus

import (
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"

	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
)

// mockDB replaces the ServiceBus DB accessors used by the OperationStop path:
// DeleteUrlsByKey succeeds, IsTableEmpty reports true.
func mockDB() *gomonkey.Patches {
	patches := gomonkey.ApplyMethod(reflect.TypeOf(&dbclient.ServiceBusService{}),
		"DeleteUrlsByKey", func(*dbclient.ServiceBusService, string) error { return nil })
	patches.ApplyMethod(reflect.TypeOf(&dbclient.ServiceBusService{}),
		"IsTableEmpty", func(*dbclient.ServiceBusService) bool { return true })
	return patches
}

func newStopMessage() *beehiveModel.Message {
	return beehiveModel.NewMessage("").
		BuildRouter(sourceType, "", "http://127.0.0.1/test", message.OperationStop)
}

// Note: like all tests in this repo that use gomonkey, run with
// `go test -gcflags "all=-N -l"` (see hack/lib/golang.sh) — gomonkey cannot
// patch inlined functions.

// Case 1: no receiver on the stop channel. The goroutine running
// processMessage must return: the fix skips the send instead of blocking
// forever, which leaked one goroutine per Stop message.
func TestProcessMessageStopNoReceiver(t *testing.T) {
	patches := mockDB()
	defer patches.Reset()

	done := make(chan struct{})
	go func() {
		processMessage(newStopMessage())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("processMessage blocked forever sending on the stop channel " +
			"with no server waiting (goroutine leak on OperationStop)")
	}
}

// Case 2: a receiver is waiting on the stop channel (as server()'s shutdown
// goroutine does). The signal must be delivered, so the normal shutdown path
// is not broken. The 50ms sleep lets the receiver goroutine park on <-c before
// processMessage runs its non-blocking send.
func TestProcessMessageStopWithReceiver(t *testing.T) {
	patches := mockDB()
	defer patches.Reset()

	received := make(chan struct{})
	go func() {
		<-c
		close(received)
	}()
	// give the receiver goroutine time to park on <-c
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		processMessage(newStopMessage())
		close(done)
	}()

	select {
	case <-received:
	case <-time.After(1 * time.Second):
		t.Fatal("stop signal was not delivered to a waiting receiver")
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("processMessage did not return after delivering the stop signal")
	}
}
