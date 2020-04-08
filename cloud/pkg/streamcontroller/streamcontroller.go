package streamcontroller

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamcontroller/constants"
)

type streamController struct {
	enable bool
}

func newstreamController(enable bool) *streamController {
	return &streamController{
		enable: enable,
	}
}

func Register() {
	core.Register(newstreamController(false))
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

	server := newServer(ts)
	server.Start()
}

func (s *streamController) Enable() bool {
	return s.enable
}
