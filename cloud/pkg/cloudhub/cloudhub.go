package cloudhub

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
	"github.com/kubeedge/viaduct/pkg/api"
)

type cloudHub struct {
	enable bool
}

func newCloudHub(enable bool) *cloudHub {
	return &cloudHub{
		enable: enable,
	}
}

func Register(hub *v1alpha1.CloudHub) {
	hubconfig.InitConfigure(hub)
	core.Register(newCloudHub(hub.Enable))
}

func (a *cloudHub) Name() string {
	return "cloudhub"
}

func (a *cloudHub) Group() string {
	return "cloudhub"
}

func (a *cloudHub) Enable() bool {
	return a.enable
}
func (a *cloudHub) Start() {
	messageq := channelq.NewChannelMessageQueue()

	// start dispatch message from the cloud to edge node
	go messageq.DispatchMessage()

	// start the cloudhub server
	if hubconfig.Get().WebSocket.Enable {
		// TODO delete second param  @kadisi
		go servers.StartCloudHub(api.ProtocolTypeWS, hubconfig.Get(), messageq)
	}

	if hubconfig.Get().Quic.Enable {
		// TODO delete second param  @kadisi
		go servers.StartCloudHub(api.ProtocolTypeQuic, hubconfig.Get(), messageq)
	}

	if hubconfig.Get().UnixSocket.Enable {
		// The uds server is only used to communicate with csi driver from kubeedge on cloud.
		// It is not used to communicate between cloud and edge.
		go udsserver.StartServer(hubconfig.Get().UnixSocket.Address)
	}
}
