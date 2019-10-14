package config

import (
	"path"

	"github.com/kubeedge/kubeedge/common/constants"
)

func NewDefaultCloudCoreConfig() *CloudCoreConfig {
	return &CloudCoreConfig{
		Kube:              NewDefaultKubeConfig(),
		EdgeController:    NewDefaultEdgeControllerConfig(),
		DeviceController:  NewDeviceControllerConfig(),
		Cloudhub:          NewDefaultCloudHubConfig(),
		Modules:           NewDefaultModules(),
		ControllerContext: NewControllerContext(),
	}
}

func NewDefaultEdgeControllerConfig() *EdgeControllerConfig {
	return &EdgeControllerConfig{
		NodeUpdateFrequency: 10,
	}
}

func NewDeviceControllerConfig() *DeviceControllerConfig {
	return &DeviceControllerConfig{}
}

func NewDefaultCloudHubConfig() *CloudHubConfig {
	return &CloudHubConfig{
		EnableWebsocket:    true,
		WebsocketPort:      10000,
		EnableQuic:         false,
		QuicPort:           10001,
		MaxIncomingStreams: 10000,
		EnableUnixSocket:   true,
		UnixSocketAddress:  "unix:///var/lib/kubeedge/kubeedge.sock",
		Address:            "0.0.0.0",
		TLSCaFile:          path.Join(constants.DefaultCADir, "rootCA.crt"),
		TLSCertFile:        path.Join(constants.DefaultCertDir, "edge.crt"),
		TLSPrivateKeyFile:  path.Join(constants.DefaultCertDir, "edge.key"),
		KeepaliveInterval:  30,
		WriteTimeout:       30,
		NodeLimit:          10,
	}
}

func NewDefaultKubeConfig() *KubeConfig {
	return &KubeConfig{
		Master:     "",
		Kubeconfig: "/root/.kube/config",
	}
}

func NewDefaultModules() *Modules {
	return &Modules{
		Enabled: []string{"devicecontroller", "edgecontroller", "cloudhub"},
	}
}

func NewDefaultAdmissionControllerConfig() *AdmissionControllerConfig {
	return &AdmissionControllerConfig{}
}

func NewControllerContext() *ControllerContext {
	return &ControllerContext{
		SendModule:     constants.DefaultContextSendModuleName,
		ReceiveModule:  constants.DefaultContextReceiveModuleName,
		ResponseModule: constants.DefaultContextResponseModuleName,
	}
}
