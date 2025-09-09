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
package edgehub

import (
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
)

func TestClassifyPriorityEdge_Rules(t *testing.T) {
	// delete -> important
	m1 := model.NewMessage("").BuildRouter("src", "grp", "res", model.DeleteOperation)
	classifyPriorityEdge(m1)
	if m1.GetPriority() != model.PriorityImportant {
		t.Fatalf("delete should be important, got %d", m1.GetPriority())
	}
	// keepalive -> important
	m2 := model.NewMessage("").BuildRouter("src", "grp", "res", messagepkg.OperationKeepalive)
	classifyPriorityEdge(m2)
	if m2.GetPriority() != model.PriorityUrgent {
		t.Fatalf("keepalive should be important, got %d", m2.GetPriority())
	}
	// response keeps
	req := model.NewMessage("").SetPriority(model.PriorityUrgent)
	resp := new(model.Message).NewRespByMessage(req, "ok")
	classifyPriorityEdge(resp)
	if resp.GetPriority() != model.PriorityUrgent {
		t.Fatalf("response should keep priority, got %d", resp.GetPriority())
	}
	// default -> normal
	m3 := model.NewMessage("").BuildRouter("src", "grp", "res", model.UpdateOperation)
	classifyPriorityEdge(m3)
	if m3.GetPriority() != model.PriorityNormal {
		t.Fatalf("default should be normal, got %d", m3.GetPriority())
	}
}
