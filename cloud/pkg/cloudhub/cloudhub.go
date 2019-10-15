package cloudhub

import (
	"context"

	"github.com/kubeedge/beehive/pkg/core"
	bcontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
	cloudconfig "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
)

type cloudHub struct {
	context *bcontext.Context
	cancel  context.CancelFunc
}

func Register(c *cloudconfig.CloudHubConfig) {
	config.InitHubConfig(c)
	core.Register(&cloudHub{})
}

func (a *cloudHub) Name() string {
	return "cloudhub"
}

func (a *cloudHub) Group() string {
	return "cloudhub"
}

func (a *cloudHub) Start(c *bcontext.Context) {
	a.context = c

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	eventq := channelq.NewChannelEventQueue(c)

	// start dispatch message from the cloud to edge node
	go eventq.DispatchMessage(ctx)

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
}

func (a *cloudHub) Cleanup() {
	a.cancel()
	a.context.Cleanup(a.Name())
}
