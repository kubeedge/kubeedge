package framework

import (
	"k8s.io/klog/v2"

	cloudconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	edgeconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

func DisableAllModules(i interface{}) {
	switch config := i.(type) {
	case *cloudconfig.CloudCoreConfig:
		config.Modules.EdgeController.Enable = false
		config.Modules.Router.Enable = false
		config.Modules.DynamicController.Enable = false
		config.Modules.CloudHub.Enable = false
		config.Modules.CloudStream.Enable = false
		config.Modules.SyncController.Enable = false
	case *edgeconfig.EdgeCoreConfig:
		config.Modules.Edged.Enable = false
		config.Modules.DBTest.Enable = false
		config.Modules.DeviceTwin.Enable = false
		config.Modules.EdgeHub.Enable = false
		config.Modules.EdgeStream.Enable = false
		config.Modules.EventBus.Enable = false
		config.Modules.MetaManager.Enable = false
		config.Modules.ServiceBus.Enable = false
		config.Modules.MetaManager.MetaServer.Enable = false
	default:
		klog.Fatal("unsupport config type")
	}
}
