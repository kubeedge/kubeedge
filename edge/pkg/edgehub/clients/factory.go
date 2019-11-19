package clients

import (
	"errors"

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
func GetClient(clientType string, config config.Configure) (Adapter, error) {

	switch clientType {
	case ClientTypeWebSocket:
		websocketConf := wsclient.WebSocketConfig{
			URL:              config.WSConfig.URL,
			CertFilePath:     config.WSConfig.CertFilePath,
			KeyFilePath:      config.WSConfig.KeyFilePath,
			HandshakeTimeout: config.WSConfig.HandshakeTimeout,
			ReadDeadline:     config.WSConfig.ReadDeadline,
			WriteDeadline:    config.WSConfig.WriteDeadline,
			ProjectID:        config.CtrConfig.ProjectID,
			NodeID:           config.CtrConfig.NodeID,
		}
		return wsclient.NewWebSocketClient(&websocketConf), nil
	case ClientTypeQuic:
		quicConfig := quicclient.QuicConfig{
			Addr:             config.QcConfig.URL,
			CaFilePath:       config.QcConfig.CaFilePath,
			CertFilePath:     config.QcConfig.CertFilePath,
			KeyFilePath:      config.QcConfig.KeyFilePath,
			HandshakeTimeout: config.QcConfig.HandshakeTimeout,
			ReadDeadline:     config.QcConfig.ReadDeadline,
			WriteDeadline:    config.QcConfig.WriteDeadline,
			ProjectID:        config.CtrConfig.ProjectID,
			NodeID:           config.CtrConfig.NodeID,
		}
		return quicclient.NewQuicClient(&quicConfig), nil
	default:
		klog.Errorf("Client type: %s is not supported", clientType)
	}

	return nil, ErrorWrongClientType
}
