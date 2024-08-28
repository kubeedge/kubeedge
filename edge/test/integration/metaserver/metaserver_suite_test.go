package metaserver_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	edgeconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/common"
	"github.com/kubeedge/kubeedge/edge/test/integration/utils/edge"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
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
		c.FeatureGates = make(map[string]bool, 1)
		c.FeatureGates[string(kefeatures.RequireAuthorization)] = false
		c.Modules.MetaManager.MetaServer.TLSCaFile = "/tmp/edgecore/rootCA.crt"
		c.Modules.MetaManager.MetaServer.TLSCertFile = "/tmp/edgecore/kubeedge.crt"
		c.Modules.MetaManager.MetaServer.TLSPrivateKeyFile = "/tmp/edgecore/kubeedge.key"

		Expect(utils.CfgToFile(c)).Should(BeNil())
		Expect(utils.StartEdgeCore()).Should(BeNil())
	})
	AfterSuite(func() {
		By("After Suite Execution....!")
	})

	RunSpecs(t, "kubeedge metaserver Suite")
}
