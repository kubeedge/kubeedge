package clients

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients/wsclient"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

//constant for reference to web socket of client
const (
	ClientTypeWebSocket = "websocket"
)

//GetClient returns an Adapter object with new web socket
func GetClient(clientType string, config *config.EdgeHubConfig) Adapter {
	if clientType == ClientTypeWebSocket {
		websocketConf := wsclient.WebSocketConfig{
			URL:              config.WSConfig.URL,
			CertFilePath:     config.WSConfig.CertFilePath,
			KeyFilePath:      config.WSConfig.KeyFilePath,
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
