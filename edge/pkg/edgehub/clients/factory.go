package clients

import (
	"errors"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients/quicclient"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients/wsclient"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

//constant for reference to web socket of client
const (
	ClientTypeWebSocket = "websocket"
	ClientTypeQuic      = "quic"
)

// ErrorWrongClientType is Wrong Client Type Error
var ErrorWrongClientType = errors.New("wrong Client Type")

//GetClient returns an Adapter object with new web socket
func GetClient(clientType string, config *config.Config) (Adapter, error) {

	switch clientType {
	case ClientTypeWebSocket:
		websocketConf := wsclient.WebSocketConfig{
			URL:              config.WebSocketURL,
			CertFilePath:     config.EdgeHubConfig.WebSocket.TLSCertFile,
			KeyFilePath:      config.EdgeHubConfig.WebSocket.TLSPrivateKeyFile,
			HandshakeTimeout: time.Duration(config.EdgeHubConfig.WebSocket.HandshakeTimeout) * time.Second,
			ReadDeadline:     time.Duration(config.EdgeHubConfig.WebSocket.ReadDeadline) * time.Second,
			WriteDeadline:    time.Duration(config.EdgeHubConfig.WebSocket.WriteDeadline) * time.Second,
			ProjectID:        config.EdgeHubConfig.Controller.ProjectId,
			NodeID:           config.EdgedConfig.HostnameOverride,
		}
		return wsclient.NewWebSocketClient(&websocketConf), nil
	case ClientTypeQuic:
		quicConfig := quicclient.QuicConfig{
			Addr:             config.EdgeHubConfig.Quic.Server,
			CaFilePath:       config.EdgeHubConfig.Quic.TLSCaFile,
			CertFilePath:     config.EdgeHubConfig.Quic.TLSCertFile,
			KeyFilePath:      config.EdgeHubConfig.Quic.TLSPrivateKeyFile,
			HandshakeTimeout: time.Duration(config.EdgeHubConfig.Quic.HandshakeTimeout) * time.Second,
			ReadDeadline:     time.Duration(config.EdgeHubConfig.Quic.ReadDeadline) * time.Second,
			WriteDeadline:    time.Duration(config.EdgeHubConfig.Quic.WriteDeadline) * time.Second,
			ProjectID:        config.EdgeHubConfig.Controller.ProjectId,
			NodeID:           config.EdgedConfig.HostnameOverride,
		}
		return quicclient.NewQuicClient(&quicConfig), nil
	default:
		klog.Errorf("Client type: %s is not supported", clientType)
	}

	return nil, ErrorWrongClientType
}
