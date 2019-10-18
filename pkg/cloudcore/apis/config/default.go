package config

import (
	"path"

	"github.com/kubeedge/kubeedge/common/constants"
	commonconfig "github.com/kubeedge/kubeedge/pkg/common/apis/config"
)

func NewDefaultCloudCoreConfig() *CloudCoreConfig {
	return &CloudCoreConfig{
		Kube:             NewDefaultKubeConfig(),
		EdgeController:   NewDefaultEdgeControllerConfig(),
		DeviceController: NewDeviceControllerConfig(),
		Cloudhub:         NewDefaultCloudHubConfig(),
		Modules:          NewDefaultModules(),
	}
}

func NewDefaultEdgeControllerConfig() *EdgeControllerConfig {
	return &EdgeControllerConfig{
		NodeUpdateFrequency: 10,
		ControllerContext: &ControllerContext{
			SendModule:     constants.CloudHubControllerModuleName,
			ReceiveModule:  constants.EdgeControllerModuleName,
			ResponseModule: constants.CloudHubControllerModuleName,
		},
	}
}

func NewDeviceControllerConfig() *DeviceControllerConfig {
	return &DeviceControllerConfig{
		ControllerContext: &ControllerContext{
			SendModule:     constants.CloudHubControllerModuleName,
			ReceiveModule:  constants.DeviceControllerModuleName,
			ResponseModule: constants.CloudHubControllerModuleName,
		},
	}
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

func NewDefaultModules() *commonconfig.Modules {
	return &commonconfig.Modules{
		Enabled: []string{"devicecontroller", "edgecontroller", "cloudhub"},
	}
}

func NewDefaultAdmissionControllerConfig() *AdmissionControllerConfig {
	return &AdmissionControllerConfig{}
}
