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

package metamanager

import (
	"testing"
	"time"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	commodule "github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	metamanagerconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
)

// TestProcessRemoteErrorCorrelatesWithCaller verifies that when a remote query
// to the cloud fails, the error response is correlated back to the original
// caller. beehive SendResp matches a sync response to its caller by parent ID,
// and processRemote regenerates the message ID before contacting the cloud, so
// the error response must be built against the original request ID. Otherwise
// the response is dropped and the caller times out instead of failing fast.
func TestProcessRemoteErrorCorrelatesWithCaller(t *testing.T) {
	// Point the remote at a module that does not exist so the cloud-side
	// SendSync fails immediately, as it does on a cloudhub outage.
	metamanagerconfig.Config.ContextSendModule = "nonexistent-cloudhub"
	metamanagerconfig.Config.RemoteQueryTimeout = 1

	m := &metaManager{enable: true}

	// Simulate the metamanager receive loop for exactly one request.
	go func() {
		msg, err := beehiveContext.Receive(commodule.MetaManagerModuleName)
		if err != nil {
			t.Errorf("failed to receive message: %v", err)
			return
		}
		m.processRemote(msg)
	}()

	// Simulate the caller in edge/pkg/metamanager/client/metaclient.go
	// (e.g. secrets.Get remote query) waiting synchronously for a response.
	msg := model.NewMessage("").BuildRouter(
		commodule.EdgedModuleName, commodule.MetaGroup,
		"default/"+model.ResourceTypeSecret+"/testsecret", model.QueryOperation)

	start := time.Now()
	resp, err := beehiveContext.SendSync(commodule.MetaManagerModuleName, *msg, 3*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("caller timed out after %v: the transport-error response was not "+
			"correlated to the original request (parent ID points at the regenerated "+
			"message ID, so beehive SendResp drops the response)", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Errorf("error response arrived after %v, want a fast failure", elapsed)
	}
	if resp.GetOperation() != model.ResponseErrorOperation {
		t.Errorf("response operation = %q, want %q", resp.GetOperation(), model.ResponseErrorOperation)
	}
	if resp.GetParentID() != msg.GetID() {
		t.Errorf("response parent ID = %q, want the original request ID %q", resp.GetParentID(), msg.GetID())
	}
}
