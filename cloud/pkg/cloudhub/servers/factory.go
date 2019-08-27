package servers

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/quicserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/udsserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/wsserver"
)

const (
	ProtocolWebsocket = "websocket"
	ProtocolQuic      = "quic"
	ProtocolUDS       = "uds"
)

func StartCloudHub(protocol string, eventq *channelq.ChannelEventQueue, c *context.Context) {
	switch protocol {
	case ProtocolWebsocket:
		wsserver.StartCloudHub(util.HubConfig, eventq, c)
	case ProtocolQuic:
		quicserver.StartCloudHub(util.HubConfig, eventq, c)
	case ProtocolUDS:
		udsserver.StartServer(util.HubConfig, c)
	default:
		panic(fmt.Errorf("invalid protocol, should be websocket or quic or uds"))
	}
}
