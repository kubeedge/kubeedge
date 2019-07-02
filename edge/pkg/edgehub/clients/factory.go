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

// Package clients
package clients

import (
	"errors"

	"github.com/kubeedge/beehive/pkg/common/log"
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
func GetClient(clientType string, config *config.EdgeHubConfig) (Adapter, error) {

	switch clientType {
	case ClientTypeWebSocket:
		websocketConf := wsclient.WebSocketConfig{
			URL:              config.WSConfig.URL,
			CertFilePath:     config.WSConfig.CertFilePath,
			KeyFilePath:      config.WSConfig.KeyFilePath,
			HandshakeTimeout: config.WSConfig.HandshakeTimeout,
			ReadDeadline:     config.WSConfig.ReadDeadline,
			WriteDeadline:    config.WSConfig.WriteDeadline,
			ExtendHeader:     config.WSConfig.ExtendHeader,
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
		log.LOGGER.Errorf("Client type: %s is not supported", clientType)
	}

	return nil, ErrorWrongClientType
}
