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

package streamrulecontroller

import (
	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	streamruleconfig "github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/listener"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/provider/streamruleendpoint"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/streamrule"
)

type StreamRuleController struct {
	enable bool
}

var _ core.Module = (*StreamRuleController)(nil)

func newStreamRuleController(enable bool) *StreamRuleController {
	return &StreamRuleController{
		enable: enable,
	}
}

func Register(src *v1alpha1.StreamRuleController) {
	streamruleconfig.InitConfigure(src)
	core.Register(newStreamRuleController(src.Enable))
}

func (src *StreamRuleController) Name() string {
	return modules.StreamRuleControllerModuleName
}

func (src *StreamRuleController) Group() string {
	return modules.StreamRuleControllerGroupName
}

func (src *StreamRuleController) Enable() bool {
	return src.enable
}

func (src *StreamRuleController) Start() {
	listener.Process(src.Name())
}
