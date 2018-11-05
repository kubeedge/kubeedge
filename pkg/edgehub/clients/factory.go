package clients

import (
	"edge-core/beehive/pkg/common/log"

	"edge-core/pkg/edgehub/clients/wsclient"
	"edge-core/pkg/edgehub/config"
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
