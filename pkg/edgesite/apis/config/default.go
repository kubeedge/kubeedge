package config

import (
	cloudcoreconfig "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
	commonconfig "github.com/kubeedge/kubeedge/pkg/common/apis/config"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

func NewDefaultEdgeSideConfig() *EdgeSideConfig {
	return &EdgeSideConfig{
		Mqtt:              edgecoreconfig.NewDefaultMqttConfig(),
		Kube:              cloudcoreconfig.NewDefaultKubeConfig(),
		ControllerContext: NewDefaultControllerContext(),
		Edged:             edgecoreconfig.NewDefaultEdgedConfig(),
		Modules:           NewDefaultModules(),
		Metamanager:       NewDefaultMetamanager(),
	}
}

func NewDefaultControllerContext() *cloudcoreconfig.ControllerContext {
	return &cloudcoreconfig.ControllerContext{
		SendModule:     "metaManager",
		ReceiveModule:  "edgecontroller",
		ResponseModule: "metaManager",
	}
}

func NewDefaultModules() *commonconfig.Modules {
	return &commonconfig.Modules{
		Enabled: []string{"edgecontroller", "metaManager", "edged", "dbTest"},
	}
}

func NewDefaultMetamanager() *Metamanager {
	return &Metamanager{
		ContextSendGroup:  "edgecontroller",
		ContextSendModule: "edgecontroller",
		EdgeSite:          true,
	}
}
