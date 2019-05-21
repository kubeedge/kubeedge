/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deployment

import (
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var (
	nodeName     string
	nodeSelector string
	//context to load config and access across the package
	ctx *utils.TestContext
)

//Function to run the Ginkgo Test
func TestEdgecoreAppDeployment(t *testing.T) {

	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		utils.InfoV6("Before Suite Execution")
		//cfg = utils.LoadConfig()
		ctx = utils.NewTestContext(utils.LoadConfig())
		nodeName = "integration-node-" + utils.GetRandomString(10)
		nodeSelector = "node-" + utils.GetRandomString(3)

		//Generate Cerificates for Edge and Cloud nodes copy to respective folders
		Expect(utils.GenerateCerts()).Should(BeNil())
		//Do the neccessary config changes in Cloud and Edge nodes
		Expect(utils.DeploySetup(ctx, nodeName, "deployment")).Should(BeNil())
		//Run ./edgecontroller binary
		Expect(utils.StartEdgeController()).Should(BeNil())
		//Register the Edge Node to Master
		Expect(utils.RegisterNodeToMaster(nodeName, ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, nodeSelector)).Should(BeNil())
		//Run ./edge_core after node registration
		Expect(utils.StartEdgeCore()).Should(BeNil())
		//Check node successfully registered or not
		Eventually(func() string {
			status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, nodeName)
			utils.Info("Node Name: %v, Node Status: %v", nodeName, status)
			return status
		}, "60s", "4s").Should(Equal("Running"), "Node register to the k8s master is unsuccessfull !!")

	})
	AfterSuite(func() {
		By("After Suite Execution....!")
		//Deregister the edge node from master
		Expect(utils.DeRegisterNodeFromMaster(ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, nodeName)).Should(BeNil())
		Eventually(func() int {
			statuscode := utils.CheckNodeDeleteStatus(ctx.Cfg.K8SMasterForKubeEdge+constants.NodeHandler, nodeName)
			utils.Info("Node Name: %v, Node Statuscode: %v", nodeName, statuscode)
			return statuscode
		}, "60s", "4s").Should(Equal(http.StatusNotFound), "Node register to the k8s master is unsuccessfull !!")
		//Run the Cleanup steps to kill edge_core and edgecontroller binaries
		Expect(utils.CleanUp("deployment")).Should(BeNil())
		//time.Sleep(2 * time.Second)
		utils.Info("Cleanup is Successfull !!")
	})

	RunSpecs(t, "kubeedge App Deploymet Suite")
}
