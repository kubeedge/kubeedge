package metaserver

import (
	"testing"

	cloudconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	edgeconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/tests/integration/framework"
)

func TestMain(m *testing.M) {
	//framework.RunCloud(modifyCloudCoreConfig)
	//framework.RunEdge(modifyEdgeCoreConfig)
	m.Run()
}

func modifyCloudCoreConfig(config *cloudconfig.CloudCoreConfig) {
	framework.DisableAllModules(config)
	config.Modules.CloudHub.Enabled = true
	config.Modules.SyncController.Enabled = true
}

func modifyEdgeCoreConfig(config *edgeconfig.EdgeCoreConfig) {
	framework.DisableAllModules(config)
	config.Modules.EdgeHub.Enabled = true
	config.Modules.MetaManager.Enabled = true
	config.Modules.MetaManager.MetaServer.Enable = true
}
