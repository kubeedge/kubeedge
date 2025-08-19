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
package dispatcher

import (
	"strings"
	"testing"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
)

func TestClassifyPriority_Rules(t *testing.T) {
	// delete -> important
	m1 := beehivemodel.NewMessage("").SetResourceOperation("node/x/res", beehivemodel.DeleteOperation)
	classifyPriority(m1)
	if m1.GetPriority() != beehivemodel.PriorityImportant {
		t.Fatalf("delete should be important, got %d", m1.GetPriority())
	}
	// podstatus -> important
	m2 := beehivemodel.NewMessage("").SetResourceOperation("node/x/"+beehivemodel.ResourceTypePodStatus, beehivemodel.UpdateOperation)
	classifyPriority(m2)
	if m2.GetPriority() != beehivemodel.PriorityImportant {
		t.Fatalf("podstatus should be important, got %d", m2.GetPriority())
	}
	if !strings.Contains(m2.GetResource(), beehivemodel.ResourceTypePodStatus) {
		t.Fatalf("malformed resource for podstatus test: %s", m2.GetResource())
	}
	// upload -> low
	m3 := beehivemodel.NewMessage("").SetResourceOperation("node/x/res", beehivemodel.UploadOperation)
	classifyPriority(m3)
	if m3.GetPriority() != beehivemodel.PriorityLow {
		t.Fatalf("upload should be low, got %d", m3.GetPriority())
	}
	// keepalive -> important
	m4 := beehivemodel.NewMessage("").SetResourceOperation("node/x/res", model.OpKeepalive)
	classifyPriority(m4)
	if m4.GetPriority() != beehivemodel.PriorityImportant {
		t.Fatalf("keepalive should be important, got %d", m4.GetPriority())
	}
	// response keeps
	req := beehivemodel.NewMessage("").SetPriority(beehivemodel.PriorityUrgent)
	resp := new(beehivemodel.Message).NewRespByMessage(req, "ok")
	classifyPriority(resp)
	if resp.GetPriority() != beehivemodel.PriorityUrgent {
		t.Fatalf("response should keep priority, got %d", resp.GetPriority())
	}
	// default -> normal
	m5 := beehivemodel.NewMessage("").SetResourceOperation("node/x/res", beehivemodel.UpdateOperation)
	classifyPriority(m5)
	if m5.GetPriority() != beehivemodel.PriorityNormal {
		t.Fatalf("default should be normal, got %d", m5.GetPriority())
	}
}
