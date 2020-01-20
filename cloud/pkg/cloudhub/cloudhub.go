package cloudhub

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
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

// Enable indicates whether enable this module
func (a *cloudHub) Enable() bool {
	return a.enable
}

func (a *cloudHub) Start() {
	messageq := channelq.NewChannelMessageQueue()

	// start dispatch message from the cloud to edge node
	go messageq.DispatchMessage()

	servers.StartCloudHub(messageq)

	if hubconfig.Get().UnixSocket.Enable {
		// The uds server is only used to communicate with csi driver from kubeedge on cloud.
		// It is not used to communicate between cloud and edge.
		go udsserver.StartServer(hubconfig.Get().UnixSocket.Address)
	}
}
