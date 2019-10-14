package cloudhub

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	cloudconfig "github.com/kubeedge/kubeedge/cloud/pkg/apis/cloudcore/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
)

type cloudHub struct {
	context  *context.Context
	stopChan chan bool
}

func Register(c *cloudconfig.CloudHubConfig) {
	config.InitHubConfig(*c)
	core.Register(&cloudHub{})
}

func (a *cloudHub) Name() string {
	return "cloudhub"
}

func (a *cloudHub) Group() string {
	return "cloudhub"
}

func (a *cloudHub) Start(c *context.Context) {
	a.context = c
	a.stopChan = make(chan bool)

	eventq := channelq.NewChannelEventQueue(c)

	// start dispatch message from the cloud to edge node
	go eventq.DispatchMessage()

	// start the cloudhub server
	if config.Conf().EnableWebsocket {
		go servers.StartCloudHub(servers.ProtocolWebsocket, eventq, c)
	}

	if config.Conf().EnableQuic {
		go servers.StartCloudHub(servers.ProtocolQuic, eventq, c)
	}

	if config.Conf().EnableUnixSocket {
		// The uds server is only used to communicate with csi driver from kubeedge on cloud.
		// It is not used to communicate between cloud and edge.
		go udsserver.StartServer(config.Conf(), c)
	}

	<-a.stopChan
}

func (a *cloudHub) Cleanup() {
	a.stopChan <- true
	a.context.Cleanup(a.Name())
}
