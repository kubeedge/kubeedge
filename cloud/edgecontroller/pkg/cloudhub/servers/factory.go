package servers

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/handler"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/servers/quicserver"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub/servers/wsserver"
)

const (
	PROTOCOL_WEBSOCKET = "websocket"
	PROTOCOL_QUIC      = "quic"
)

func StartCloudHub(protocol string, eventq *channelq.ChannelEventQueue, c *context.Context) {
	if protocol == PROTOCOL_WEBSOCKET {
		wsserver.StartCloudHub(util.HubConfig, eventq)
		handler.WebSocketEventHandler.Context = c
	} else if protocol == PROTOCOL_QUIC {
		quicserver.StartCloudHub(util.HubConfig, eventq)
		handler.QuicEventHandler.Context = c
	} else {
		panic(fmt.Errorf("invalid protocol, should be websocket or quic."))
	}
}
