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
package nodedensity

import (
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
	. "github.com/kubeedge/kubeedge/tests/performance/common"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/api/core/v1"
)

var DeploymentTestTimerGroup *utils.TestTimerGroup = utils.NewTestTimerGroup()

var _ = Describe("Application deployment test in Perfronace test EdgeNodes", func() {

	Context("Test application deployment on specific EdgeNode", func() {
		var testTimer *utils.TestTimer
		var testDescription GinkgoTestDescription
		var podlist metav1.PodList
		var NoOfEdgeNodes int

		BeforeEach(func() {
			testDescription = CurrentGinkgoTestDescription()
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			DeleteEdgeDeployments(ctx.Cfg.K8SMasterForKubeEdge, ctx.Cfg.K8SMasterForProvisionEdgeNodes, NoOfEdgeNodes)
			utils.CheckDeploymentPodDeleteState(ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler, podlist)
		})

		Measure("PERF_NODETEST_NODES_1: Create 1 KubeEdge Node Deployment, Measure Node Ready time", func(b Benchmarker) {
			NoOfEdgeNodes = 1
			//Deploye edgecore as a k8s resource to cluster-2
			podlist = HandleEdgeDeployment(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler,
				ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler, NoOfEdgeNodes)
			runtime := b.Time("PERF_NODETEST_NODES_1", func() {
				//Register EdgeNodes to K8s Master
				for edgenodeName, val := range NodeInfo {
					utils.RegisterNodeToMaster(edgenodeName, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, val[1])
				}
				time.Sleep(1 * time.Second)
				//Check All EdgeNode are in Running state
				Eventually(func() int {
					count := 0
					for edgenodeName := range NodeInfo {
						status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, edgenodeName)
						utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
						if status == "Running" {
							count++
						}
					}
					return count
				}, "300s", "2s").Should(Equal(NoOfEdgeNodes), "Nodes register to the k8s master is unsuccessfull !!")

			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
		Measure("PERF_NODETEST_NODES_20: Create 5 KubeEdge Node Deployment, Measure Node Ready time", func(b Benchmarker) {
			NoOfEdgeNodes = 20
			//Deploye edgecore as a k8s resource to cluster-2
			podlist = HandleEdgeDeployment(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler,
				ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler, NoOfEdgeNodes)
			runtime := b.Time("PERF_NODETEST_NODES_1", func() {
				//Register EdgeNodes to K8s Master
				for edgenodeName, val := range NodeInfo {
					utils.RegisterNodeToMaster(edgenodeName, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, val[1])
				}
				Eventually(func() int {
					count := 0
					for edgenodeName := range NodeInfo {
						status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, edgenodeName)
						utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
						if status == "Running" {
							count++
						}
					}
					return count
				}, "300s", "2s").Should(Equal(NoOfEdgeNodes), "Nodes register to the k8s master is unsuccessfull !!")
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)

		Measure("PERF_NODETEST_NODES_50: Create 10 KubeEdge Node Deployment, Measure Node Ready time", func(b Benchmarker) {
			NoOfEdgeNodes = 50
			//Deploye edgecore as a k8s resource to cluster-2
			podlist = HandleEdgeDeployment(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler,
				ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler, NoOfEdgeNodes)
			runtime := b.Time("PERF_NODETEST_NODES_1", func() {
				//Register EdgeNodes to K8s Master
				for edgenodeName, val := range NodeInfo {
					utils.RegisterNodeToMaster(edgenodeName, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, val[1])
				}
				Eventually(func() int {
					count := 0
					for edgenodeName := range NodeInfo {
						status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, edgenodeName)
						utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
						if status == "Running" {
							count++
						}
					}
					return count
				}, "300s", "2s").Should(Equal(NoOfEdgeNodes), "Nodes register to the k8s master is unsuccessfull !!")
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
		Measure("PERF_NODETEST_NODES_100: Create 20 KubeEdge Node Deployment, Measure Node Ready time", func(b Benchmarker) {
			NoOfEdgeNodes = 100
			//Deploye edgecore as a k8s resource to cluster-2
			podlist = HandleEdgeDeployment(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler,
				ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler, NoOfEdgeNodes)
			runtime := b.Time("PERF_NODETEST_NODES_1", func() {
				//Register EdgeNodes to K8s Master
				for edgenodeName, val := range NodeInfo {
					utils.RegisterNodeToMaster(edgenodeName, ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, val[1])
				}
				Eventually(func() int {
					count := 0
					for edgenodeName := range NodeInfo {
						status := utils.CheckNodeReadyStatus(ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, edgenodeName)
						utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
						if status == "Running" {
							count++
						}
					}
					return count
				}, "300s", "2s").Should(Equal(NoOfEdgeNodes), "Nodes register to the k8s master is unsuccessfull !!")
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
	})
})
