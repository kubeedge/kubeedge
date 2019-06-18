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
package hubtest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
	. "github.com/kubeedge/kubeedge/tests/performance/common"
)

// configs across the package
var (
	ctx              *utils.TestContext
	cfg              utils.Config
	cloudHubURL      string
	controllerHubURL string
)

func TestKubeEdgeK8SDeployment(t *testing.T) {
	// Init params
	var podlist v1.PodList
	RegisterFailHandler(Fail)

	// Init suite
	var _ = BeforeSuite(func() {
		// Init config
		utils.InfoV6("KubeEdge hub performance test begin!")
		cfg = utils.LoadConfig()
		ctx = utils.NewTestContext(cfg)

		//apply label to all cluster nodes, use the selector to deploy all edgenodes to cluster nodes
		err := ApplyLabel(ctx.Cfg.K8SMasterForProvisionEdgeNodes + NodeHandler)
		Expect(err).Should(BeNil())

		// Deploy KubeEdge Cloud Part as a k8s deployment into KubeEdge Cluster
		CloudConfigMap = "cloudcore-configmap-" + utils.GetRandomString(5)
		CloudCoreDeployment = "cloudcore-deployment-" + utils.GetRandomString(5)
		err = HandleCloudDeployment(
			CloudConfigMap,
			CloudCoreDeployment,
			ctx.Cfg.K8SMasterForKubeEdge,
			ctx.Cfg.K8SMasterForKubeEdge+ConfigmapHandler,
			ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler,
			ctx.Cfg.CloudImageUrl,
			ctx.Cfg.NumOfNodes)
		Expect(err).Should(BeNil())
		time.Sleep(1 * time.Second)

		// Get KubeEdge Cloud Part host ip
		podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, "")
		Expect(err).To(BeNil())
		cloudPartHostIP := ""
		for _, pod := range podlist.Items {
			if strings.Contains(pod.Name, "cloudcore-deployment") {
				cloudPartHostIP = pod.Status.HostIP
				break
			}
		}

		// Check if KubeEdge Cloud Part is running
		utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
		time.Sleep(5 * time.Second)

		// Create NodePort Service for KubeEdge Cloud Part
		err = utils.ExposeCloudService(CloudCoreDeployment, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
		Expect(err).Should(BeNil())

		// Get NodePort Service to access KubeEdge Cloud Part from KubeEdge Edge Nodes
		wsPort, _ := utils.GetServicePort(CloudCoreDeployment, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
		cloudHubURL = fmt.Sprintf("wss://%s:%d", cloudPartHostIP, wsPort)
		controllerHubURL = fmt.Sprintf("http://%s:%d", cloudPartHostIP, ctx.Cfg.ControllerStubPort)
	})
	AfterSuite(func() {
		By("KubeEdge hub performance test end!")
		// Delete KubeEdge Cloud Part deployment
		DeleteCloudDeployment(ctx.Cfg.K8SMasterForKubeEdge)
		// Check if KubeEdge Cloud Part is deleted
		utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
	})
	RunSpecs(t, "KubeEdge hub performance test suite")
}
