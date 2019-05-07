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
	"os/exec"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//context to load config and access across the package
var (
	ctx          *utils.TestContext
	cfg          utils.Config
	nodeSelector string
	nodeName     string
)

var (
	runController = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/cloud; sudo nohup ./edgecontroller > edgecontroller.log 2>&1 &"
	runEdgecore   = "cd ${GOPATH}/src/github.com/kubeedge/kubeedge/edge/; sudo nohup ./edge_core > edge_core.log 2>&1 &"
)

//Function to run the Ginkgo Test
func TestEdgecoreAppDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		utils.InfoV6("Before Suite Execution")
		cfg = utils.LoadConfig()
		ctx = utils.NewTestContext(cfg)
		nodeName = "integration-node-" + utils.GetRandomString(10)
		nodeSelector = "node-" + utils.GetRandomString(3)
		//Generate Cerificates for Edge and Cloud nodes copy to respective folders
		cmd := exec.Command("bash", "-x", "scripts/generate_cert.sh")
		err := utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
		//Do the neccessary config changes in Cloud and Edge nodes
		cmd = exec.Command("bash", "-x", "scripts/setup.sh", nodeName, ctx.Cfg.K8SMasterForKubeEdge)
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
		time.Sleep(1 * time.Second)
		//Run ./edgecontroller binary
		cmd = exec.Command("sh", "-c", runController)
		err = utils.PrintCombinedOutput(cmd)
		time.Sleep(5 * time.Second)
		Expect(err).Should(BeNil())
		//Register the Edge Node to Master
		err = utils.RegisterNodeToMaster(nodeName, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, nodeSelector)
		Expect(err).Should(BeNil())
		//Run ./edge_core after node registration
		cmd = exec.Command("sh", "-c", runEdgecore)
		err = utils.PrintCombinedOutput(cmd)
		time.Sleep(5 * time.Second)
		//Check node successfully registered or not
		Eventually(func() string {
			status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, nodeName)
			utils.Info("Node Name: %v, Node Status: %v", nodeName, status)
			return status
		}, "60s", "4s").Should(Equal("Running"), "Node register to the k8s master is unsuccessfull !!")

	})
	AfterSuite(func() {
		By("After Suite Execution....!")
		//Deregister the edge node from master
		err := utils.DeRegisterNodeFromMaster(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, nodeName)
		Expect(err).Should(BeNil())
		Eventually(func() int {
			statuscode := utils.CheckNodeDeleteStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, nodeName)
			utils.Info("Node Name: %v, Node Statuscode: %v", nodeName, statuscode)
			return statuscode
		}, "60s", "4s").Should(Equal(http.StatusNotFound), "Node register to the k8s master is unsuccessfull !!")
		//Run the Cleanup steps to kill edge_core and edgecontroller binaries
		cmd := exec.Command("bash", "-x", "scripts/cleanup.sh")
		err = utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
		time.Sleep(2 * time.Second)
		utils.Info("Cleanup is Successfull !!")
	})

	RunSpecs(t, "kubeedge App Deploymet Suite")
}
