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

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
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
			podlist = metav1.PodList{}
			NoOfEdgeNodes = 1
			CreateConfigMapforEdgeCore(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler,
				ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NoOfEdgeNodes)
			runtime := b.Time("PERF_NODETEST_NODES_1", func() {
				podlist = HandleEdgeCorePodDeployment(ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler,
					ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler,
					ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NoOfEdgeNodes)
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
		Measure("PERF_NODETEST_NODES_5: Create 5 KubeEdge Node Deployment, Measure Node Ready time", func(b Benchmarker) {
			podlist = metav1.PodList{}
			NoOfEdgeNodes = 5
			CreateConfigMapforEdgeCore(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler,
				ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NoOfEdgeNodes)
			runtime := b.Time("PERF_NODETEST_NODES_1", func() {
				podlist = HandleEdgeCorePodDeployment(ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler,
					ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler,
					ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NoOfEdgeNodes)
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)

		Measure("PERF_NODETEST_NODES_10: Create 10 KubeEdge Node Deployment, Measure Node Ready time", func(b Benchmarker) {
			podlist = metav1.PodList{}
			NoOfEdgeNodes = 10
			CreateConfigMapforEdgeCore(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler,
				ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NoOfEdgeNodes)
			runtime := b.Time("PERF_NODETEST_NODES_1", func() {
				podlist = HandleEdgeCorePodDeployment(ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler,
					ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler,
					ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NoOfEdgeNodes)
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
		Measure("PERF_NODETEST_NODES_20: Create 20 KubeEdge Node Deployment, Measure Node Ready time", func(b Benchmarker) {
			podlist = metav1.PodList{}
			NoOfEdgeNodes = 50
			CreateConfigMapforEdgeCore(cloudHubURL, ctx.Cfg.K8SMasterForProvisionEdgeNodes+ConfigmapHandler,
				ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NoOfEdgeNodes)
			runtime := b.Time("PERF_NODETEST_NODES_1", func() {
				podlist = HandleEdgeCorePodDeployment(ctx.Cfg.K8SMasterForProvisionEdgeNodes+DeploymentHandler,
					ctx.Cfg.EdgeImageUrl, ctx.Cfg.K8SMasterForProvisionEdgeNodes+AppHandler,
					ctx.Cfg.K8SMasterForKubeEdge+NodeHandler, NoOfEdgeNodes)
			})
			glog.Infof("Runtime stats: %+v", runtime)
		}, 5)
	})
})
