package servers

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/quicserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/wsserver"
)

const (
	ProtocolWebsocket = "websocket"
	ProtocolQuic      = "quic"
)

func StartCloudHub(protocol string, eventq *channelq.ChannelEventQueue, c *context.Context) {
	if protocol == ProtocolWebsocket {
		wsserver.StartCloudHub(util.HubConfig, eventq)
		handler.WebSocketHandler.EventHandler.Context = c
	} else if protocol == ProtocolQuic {
		quicserver.StartCloudHub(util.HubConfig, eventq)
		handler.QuicHandler.EventHandler.Context = c
	} else {
		panic(fmt.Errorf("invalid protocol, should be websocket or quic"))
	}
}
