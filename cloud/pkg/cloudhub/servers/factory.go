package servers

import (
	"fmt"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/quicserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/wsserver"
)

const (
	ProtocolWebsocket = "websocket"
	ProtocolQuic      = "quic"
)

func StartCloudHub(protocol string, eventq *channelq.ChannelEventQueue, c *context.Context) {
	switch protocol {
	case ProtocolWebsocket:
		wsserver.StartCloudHub(config.Conf(), eventq, c)
	case ProtocolQuic:
		quicserver.StartCloudHub(config.Conf(), eventq, c)
	default:
		panic(fmt.Errorf("invalid protocol, should be websocket or quic or uds"))
	}
}
