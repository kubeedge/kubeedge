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
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
	. "github.com/kubeedge/kubeedge/tests/performance/common"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
	"github.com/kubeedge/kubeedge/tests/stubs/common/types"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/api/core/v1"
)

var _ = Describe("KubeEdge hub performance test", func() {
	Context("Test different numbers of Pods on different numbers of Edge Nodes", func() {
		// Init params
		var podlist metav1.PodList
		var numOfEdgeNodes int
		var numOfPodsPerEdgeNode int
		var podsInfo map[string]types.FakePod

		BeforeEach(func() {
			// Create Edge Nodes
			numOfEdgeNodes = 10
			podlist = HandleEdgeDeployment(cloudHubURL, ctx.Cfg.ApiServer2+DeploymentHandler, ctx.Cfg.ApiServer+NodeHandler,
				ctx.Cfg.ApiServer2+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.ApiServer2+AppHandler, numOfEdgeNodes)
		})

		AfterEach(func() {
			// Delete Pods
			for _, p := range podsInfo {
				DeleteFakePod(controllerHubURL, p)
			}
			// Check All Pods are deleted
			Eventually(func() int {
				ps := ListFakePods(controllerHubURL)
				return len(ps)
			}, "240s", "4s").Should(Equal(0), "Wait for Pods deleted timeout")

			// Delete Edge Nodes
			DeleteEdgeDeployments(ctx.Cfg.ApiServer, ctx.Cfg.ApiServer2, numOfEdgeNodes)
			utils.CheckDeploymentPodDeleteState(ctx.Cfg.ApiServer2+AppHandler, podlist)
		})

		Measure("PERF_HUBTEST_NODES_10_PODS_10: Create 10 Edge Nodes, Deploy 10 Pods per Edge Node, Measure startup time of Pods", func(b Benchmarker) {
			// Measure startup time
			runtime := b.Time("runtime", func() {
				// Create Pods on Edge Nodes
				numOfPodsPerEdgeNode = 10
				podsInfo = make(map[string]types.FakePod)
				// Loop for Pod Numbers
				for i := 0; i < numOfPodsPerEdgeNode; i++ {
					// Loop for Edge Node Numbers
					for nodeName, _ := range NodeInfo {
						// Contruct fake pods
						var pod types.FakePod
						pod.Name = nodeName + "-fakepod-" + utils.GetRandomString(10)
						pod.Namespace = constants.NamespaceDefault
						pod.NodeName = nodeName
						pod.Status = constants.PodPending
						// Add fake pod
						AddFakePod(controllerHubURL, pod)
						// Store fake pod
						podsInfo[pod.Name] = pod
					}
				}

				// Check all pods are running
				Eventually(func() int {
					count := 0
					// List all pods status
					pods := ListFakePods(controllerHubURL)
					// Get current pod numbers which are running
					for _, p := range pods {
						if p.Status == constants.PodRunning {
							count++
						}
					}
					return count
				}, "240s", "4s").Should(Equal(numOfEdgeNodes*numOfPodsPerEdgeNode), "Wait for Pods in running status timeout")

			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
	})
})
