package cloudhub

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
	"github.com/kubeedge/viaduct/pkg/api"
)

type cloudHub struct {
}

func newCloudHub() *cloudHub {
	return &cloudHub{}
}

func Register() {
	hubconfig.InitConfigure()
	core.Register(newCloudHub())
}

func (a *cloudHub) Name() string {
	return "cloudhub"
}

func (a *cloudHub) Group() string {
	return "cloudhub"
}

func (a *cloudHub) Start() {
	messageq := channelq.NewChannelMessageQueue()

	// start dispatch message from the cloud to edge node
	go messageq.DispatchMessage()

	// start the cloudhub server
	if hubconfig.Get().ProtocolWebsocket {
		go servers.StartCloudHub(api.ProtocolTypeWS, hubconfig.Get(), messageq)
	}

	if hubconfig.Get().ProtocolQuic {
		go servers.StartCloudHub(api.ProtocolTypeQuic, hubconfig.Get(), messageq)
	}

	if hubconfig.Get().ProtocolUDS {
		// The uds server is only used to communicate with csi driver from kubeedge on cloud.
		// It is not used to communicate between cloud and edge.
		go udsserver.StartServer(hubconfig.Get())
	}
}
