/*
Copyright 2022 The KubeEdge Authors.

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

package rule

import (
	"time"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/messagelayer"
)

type ExecResult struct {
	RuleID    string
	ProjectID string
	Status    string
	Error     ErrorMsg
}

type ErrorMsg struct {
	Detail    string
	Timestamp time.Time
}

var ResultChannel chan ExecResult
var StopChan chan bool

func init() {
	StopChan = make(chan bool)
	go SendMessageToController(StopChan)
}

func SendMessageToController(stop chan bool) {
	ResultChannel = make(chan ExecResult, 1024)
	for {
		select {
		case r := <-ResultChannel:
			msg := model.NewMessage("")
			resource, err := messagelayer.BuildResourceForRouter(r.ProjectID, model.ResourceTypeRuleStatus, r.RuleID)
			if err != nil {
				klog.Warningf("build message resource failed with error: %s", err)
				continue
			}
			msg.Content = r
			msg.BuildRouter(modules.RouterModuleName, constants.GroupResource, resource, model.UpdateOperation)
			beehiveContext.Send(modules.EdgeControllerModuleName, *msg)
			klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
		case _, ok := <-stop:
			if !ok {
				klog.Warningf("do stop channel is closed")
			}
		}
	}
}
