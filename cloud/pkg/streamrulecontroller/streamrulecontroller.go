package streamrulecontroller

import (
	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	streamruleconfig "github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/listener"

	// init streamrule
	_ "github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/streamrule"
	// init streamruleendpoint
	_ "github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/provider/streamruleendpoint"
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
