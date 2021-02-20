package framework

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/router"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/edgestream"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus"
	"github.com/kubeedge/kubeedge/edge/test"
	cloudconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	edgeconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

// registerModules register all the modules
func registerModules(i interface{}) {
	switch c := i.(type) {
	case cloudconfig.CloudCoreConfig:
		cloudhub.Register(c.Modules.CloudHub)
		edgecontroller.Register(c.Modules.EdgeController)
		devicecontroller.Register(c.Modules.DeviceController)
		synccontroller.Register(c.Modules.SyncController)
		cloudstream.Register(c.Modules.CloudStream)
		router.Register(c.Modules.Router)
		dynamiccontroller.Register(c.Modules.DynamicController)
	case edgeconfig.EdgeCoreConfig:
		devicetwin.Register(c.Modules.DeviceTwin, c.Modules.Edged.HostnameOverride)
		//edged.Register(c.Modules.Edged)
		edgehub.Register(c.Modules.EdgeHub, c.Modules.Edged.HostnameOverride)
		eventbus.Register(c.Modules.EventBus, c.Modules.Edged.HostnameOverride)
		//edgemesh.Register(c.Modules.EdgeMesh)
		metamanager.Register(c.Modules.MetaManager)
		servicebus.Register(c.Modules.ServiceBus)
		edgestream.Register(c.Modules.EdgeStream, c.Modules.Edged.HostnameOverride, c.Modules.Edged.NodeIP)
		test.Register(c.Modules.DBTest)
		// Note: Need to put it to the end, and wait for all models to register before executing
		dbm.InitDBConfig(c.DataBase.DriverName, c.DataBase.AliasName, c.DataBase.DataSource)
	}
}

func DisableAllModules(i interface{}) {
	switch config := i.(type) {
	case cloudconfig.CloudCoreConfig:
		config.Modules.EdgeController.Enable = false
		config.Modules.Router.Enable = false
		config.Modules.DynamicController.Enable = false
		config.Modules.CloudHub.Enable = false
		config.Modules.CloudStream.Enable = false
		config.Modules.SyncController.Enable = false
	case edgeconfig.EdgeCoreConfig:
		config.Modules.Edged.Enable = false
		config.Modules.DBTest.Enable = false
		config.Modules.DeviceTwin.Enable = false
		config.Modules.EdgeHub.Enable = false
		config.Modules.EdgeStream.Enable = false
		config.Modules.EventBus.Enable = false
		config.Modules.MetaManager.Enable = false
		config.Modules.ServiceBus.Enable = false
		config.Modules.MetaManager.MetaServer.Enable = false
		config.Modules.EdgeMesh.Enable = false
	}
}
