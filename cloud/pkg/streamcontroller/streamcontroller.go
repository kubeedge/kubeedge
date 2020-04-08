package streamcontroller

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamcontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamcontroller/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type streamController struct {
	enable bool
}

func newstreamController(enable bool) *streamController {
	return &streamController{
		enable: enable,
	}
}

func Register(controller *v1alpha1.StreamController) {
	config.InitConfigure(controller)
	core.Register(newstreamController(controller.Enable))
}

func (s *streamController) Name() string {
	return constants.StreamControllerModuleName
}

func (s *streamController) Group() string {
	return constants.StreamControllerGroupName
}

func (s *streamController) Start() {
	ts := newTunnelServer()
	ts.Start()

	server := newStreamServer(ts)
	server.Start()
}

func (s *streamController) Enable() bool {
	return s.enable
}
