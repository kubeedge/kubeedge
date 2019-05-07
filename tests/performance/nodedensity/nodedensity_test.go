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
	. "github.com/kubeedge/kubeedge/tests/performance/common"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"

	. "github.com/onsi/ginkgo"
	metav1 "k8s.io/api/core/v1"
	"fmt"
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
			DeleteEdgeDeployments(ctx.Cfg.ApiServer2, NoOfEdgeNodes)
			utils.CheckDeploymentPodDeleteState(ctx.Cfg.ApiServer2+AppHandler, podlist)
		})

		It("PERF_NODETEST_NODES_1: Create KubeEdge Node Deployment, Measure Node Ready time", func() {
			NoOfEdgeNodes=20
			podlist = metav1.PodList{}
			podlist = HandleEdgeDeployment(cloudHub, ctx.Cfg.ApiServer2+DeploymentHandler, ctx.Cfg.ApiServer2+NodeHandler,
				ctx.Cfg.ApiServer2+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.ApiServer2+AppHandler, NoOfEdgeNodes)
		})
		It("PERF_NODETEST_NODES_5: Create 5 KubeEdge Node Deployment, Measure Node Ready time", func() {
			podlist = metav1.PodList{}
			NoOfEdgeNodes=5
			podlist = HandleEdgeDeployment(cloudHub, ctx.Cfg.ApiServer2+DeploymentHandler, ctx.Cfg.ApiServer2+NodeHandler,
				ctx.Cfg.ApiServer2+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.ApiServer2+AppHandler, NoOfEdgeNodes)
		})

		It("PERF_NODETEST_NODES_5: Create 5 KubeEdge Node Deployment, Measure Node Ready time", func() {
			podlist = metav1.PodList{}
			NoOfEdgeNodes=5
			podlist = HandleEdgeDeployment(cloudHub, ctx.Cfg.ApiServer2+DeploymentHandler, ctx.Cfg.ApiServer2+NodeHandler,
				ctx.Cfg.ApiServer2+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.ApiServer2+AppHandler, NoOfEdgeNodes)
		})
		Measure("PERF_NODETEST_NODES_10: Create 10 KubeEdge Node Deployment, Measure Node Ready time", func(b Benchmarker) {
			podlist = metav1.PodList{}
			runtime := b.Time("runtime", func() {
				NoOfEdgeNodes=20
				podlist = HandleEdgeDeployment(cloudHub, ctx.Cfg.ApiServer2+DeploymentHandler, ctx.Cfg.ApiServer2+NodeHandler,
					ctx.Cfg.ApiServer2+ConfigmapHandler, ctx.Cfg.EdgeImageUrl, ctx.Cfg.ApiServer2+AppHandler, NoOfEdgeNodes)
			})

			fmt.Println(runtime.Seconds())

		}, 5)

	})
})
