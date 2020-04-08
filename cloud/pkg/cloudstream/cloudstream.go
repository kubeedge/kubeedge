package cloudstream

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type cloudStream struct {
	enable bool
}

func newCloudStream(enable bool) *cloudStream {
	return &cloudStream{
		enable: enable,
	}
}

func Register(controller *v1alpha1.StreamController) {
	config.InitConfigure(controller)
	core.Register(newCloudStream(controller.Enable))
}

func (s *cloudStream) Name() string {
	return constants.StreamControllerModuleName
}

func (s *cloudStream) Group() string {
	return constants.StreamControllerGroupName
}

func (s *cloudStream) Start() {
	ts := newTunnelServer()
	ts.Start()

	server := newStreamServer(ts)
	server.Start()
}

func (s *cloudStream) Enable() bool {
	return s.enable
}
