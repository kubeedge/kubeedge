package config

import (
	cloudcoreconfig "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
	commonconfig "github.com/kubeedge/kubeedge/pkg/common/apis/config"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

type EdgeSideConfig struct {
	Mqtt              *edgecoreconfig.MqttConfig         `json:"mqtt,omitempty"`
	Kube              *cloudcoreconfig.KubeConfig        `json:"kube,omitempty"`
	ControllerContext *cloudcoreconfig.ControllerContext `json:"controllerContext"`
	Edged             *edgecoreconfig.EdgedConfig        `json:"edged,omitempty"`
	Modules           *commonconfig.Modules              `json:"modules,omitempty"`
	Metamanager       *edgecoreconfig.Metamanager        `json:"metamanager,omitempty"`
}
