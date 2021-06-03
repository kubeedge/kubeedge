package metaserver_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubeedge/kubeedge/edge/test/integration/utils"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/edge"
	edgeconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/tests/integration/framework"
)

func TestEdgecoreMetaServer(t *testing.T) {
	RegisterFailHandler(Fail)
	//var UID string
	var _ = BeforeSuite(func() {
		common.Infof("Before Suite Execution")
		cfg := edge.LoadConfig()

		c := edgeconfig.NewDefaultEdgeCoreConfig()
		framework.DisableAllModules(c)
		c.Modules.Edged.HostnameOverride = cfg.NodeID
		c.Modules.MetaManager.Enable = true
		c.Modules.MetaManager.MetaServer.Enable = true

		Expect(utils.CfgToFile(c)).Should(BeNil())
		Expect(utils.StartEdgeCore()).Should(BeNil())
	})
	AfterSuite(func() {
		By("After Suite Execution....!")
	})

	RunSpecs(t, "kubeedge metaserver Suite")
}
