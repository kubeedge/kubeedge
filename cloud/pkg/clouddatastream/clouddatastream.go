package clouddatastream

import (
	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/clouddatastream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
)

type cloudDataStream struct {
	enable bool

	tunnelPort int
}

var _ core.Module = (*cloudDataStream)(nil)

func newCloudDataStream(enable bool, tunnelPort int) *cloudDataStream {
	return &cloudDataStream{
		enable: enable,

		tunnelPort: tunnelPort,
	}
}

func Register(controller *v1alpha1.CloudDataStream, commonConfig *v1alpha1.CommonConfig) {
	config.InitConfigure(controller)
	core.Register(newCloudDataStream(controller.Enable, commonConfig.TunnelPort))
}

func (s *cloudDataStream) Name() string {
	return modules.CloudDataStreamModuleName
}

func (s *cloudDataStream) Group() string {
	return modules.CloudDataStreamGroupName
}

func (s *cloudDataStream) Start() {
	ok := <-cloudhub.DoneTLSTunnelCerts

	if ok {
		ts := newTunnelServer(s.tunnelPort)

		go ts.Start()

		server := newStreamServer(ts)
		go server.Start()
	}
}

func (s *cloudDataStream) Enable() bool {
	return s.enable
}
