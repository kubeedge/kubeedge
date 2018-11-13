package clients

import (
	"github.com/kubeedge/kubeedge/beehive/pkg/common/log"

	"github.com/kubeedge/kubeedge/pkg/edgehub/clients/wsclient"
	"github.com/kubeedge/kubeedge/pkg/edgehub/config"
)

const (
	ClientTypeWebSocket = "websocket"
)

func GetClient(clientType string, config *config.EdgeHubConfig) Adapter {
	if clientType == ClientTypeWebSocket {
		websocketConf := wsclient.WebSocketConfig{
			Url:              config.WSConfig.Url,
			HandshakeTimeout: config.WSConfig.HandshakeTimeout,
			ReadDeadline:     config.WSConfig.ReadDeadline,
			WriteDeadline:    config.WSConfig.WriteDeadline,
			ExtendHeader:     config.WSConfig.ExtendHeader,
		}
		return wsclient.NewWebSocketClient(&websocketConf)
	}

	log.LOGGER.Errorf("donot support client type: %s", clientType)
	return nil
}
