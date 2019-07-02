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

// Package util
package util

import (
	"github.com/kubeedge/beehive/pkg/common/config"
)

// HubConfig is the config for entire CloudHub
var HubConfig *Config

func init() {
	HubConfig = &Config{}
	HubConfig.ProtocolWebsocket, _ = config.CONFIG.GetValue("cloudhub.protocol_websocket").ToBool()
	HubConfig.ProtocolQuic, _ = config.CONFIG.GetValue("cloudhub.protocol_quic").ToBool()
	if !HubConfig.ProtocolWebsocket && !HubConfig.ProtocolQuic {
		HubConfig.ProtocolWebsocket = true
	}

	HubConfig.Address, _ = config.CONFIG.GetValue("cloudhub.address").ToString()
	HubConfig.Port, _ = config.CONFIG.GetValue("cloudhub.port").ToInt()
	HubConfig.QuicPort, _ = config.CONFIG.GetValue("cloudhub.quic_port").ToInt()
	HubConfig.MaxIncomingStreams, _ = config.CONFIG.GetValue("cloudhub.max_incomingstreams").ToInt()
	HubConfig.KeepaliveInterval, _ = config.CONFIG.GetValue("cloudhub.keepalive-interval").ToInt()
	HubConfig.WriteTimeout, _ = config.CONFIG.GetValue("cloudhub.write-timeout").ToInt()
	HubConfig.NodeLimit, _ = config.CONFIG.GetValue("cloudhub.node-limit").ToInt()
}

// Config represents configuration options for http access
type Config struct {
	ProtocolWebsocket  bool
	ProtocolQuic       bool
	MaxIncomingStreams int
	Address            string
	Port               int
	QuicPort           int
	KeepaliveInterval  int
	Ca                 []byte
	Cert               []byte
	Key                []byte
	WriteTimeout       int
	NodeLimit          int
}
