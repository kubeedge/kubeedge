/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package servers

import (
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
	switch protocol {
	case ProtocolWebsocket:
		wsserver.StartCloudHub(util.HubConfig, eventq)
		handler.WebSocketHandler.EventHandler.Context = c
	case ProtocolQuic:
		quicserver.StartCloudHub(util.HubConfig, eventq)
		handler.QuicHandler.EventHandler.Context = c
	default:
		panic("invalid protocol, should be websocket or quic")
	}
}
