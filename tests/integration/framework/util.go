package framework

import (
	"k8s.io/klog/v2"

	cloudconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	edgeconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

func DisableAllModules(i interface{}) {
	switch config := i.(type) {
	case *cloudconfig.CloudCoreConfig:
		config.Modules.EdgeController.Enabled = false
		config.Modules.Router.Enabled = false
		config.Modules.DynamicController.Enabled = false
		config.Modules.CloudHub.Enabled = false
		config.Modules.CloudStream.Enabled = false
		config.Modules.SyncController.Enabled = false
	case *edgeconfig.EdgeCoreConfig:
		config.Modules.Edged.Enabled = false
		config.Modules.DBTest.Enable = false
		config.Modules.DeviceTwin.Enabled = false
		config.Modules.EdgeHub.Enabled = false
		config.Modules.EdgeStream.Enabled = false
		config.Modules.EventBus.Enabled = false
		config.Modules.MetaManager.Enabled = false
		config.Modules.ServiceBus.Enabled = false
		config.Modules.MetaManager.MetaServer.Enable = false
	default:
		klog.Fatal("unsupport config type")
	}
}
