package servers

import (
	"fmt"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/quicserver"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/wsserver"
)

const (
	ProtocolWebsocket = "websocket"
	ProtocolQuic      = "quic"
)

func StartCloudHub(protocol string, eventq *channelq.ChannelMessageQueue) {
	switch protocol {
	case ProtocolWebsocket:
		wsserver.StartCloudHub(util.HubConfig, eventq)
	case ProtocolQuic:
		quicserver.StartCloudHub(util.HubConfig, eventq)
	default:
		panic(fmt.Errorf("invalid protocol, should be websocket or quic or uds"))
	}
}
