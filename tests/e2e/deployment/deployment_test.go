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

	"github.com/kubeedge/kubeedge/tests/e2e/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/api/core/v1"
)

const (
	AppHandler        = "/api/v1/namespaces/default/pods"
	NodeHandler       = "/api/v1/nodes"
	DeploymentHandler = "/apis/apps/v1/namespaces/default/deployments"
)

//Run Test cases
var _ = Describe("Application deployment in edge_core Testing", func() {
	var UID string
	Context("Test application deployment and delete deployment", func() {
		BeforeEach(func() {
		})
		AfterEach(func() {
			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.ApiServer+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				label := deployment.ObjectMeta.Labels["app"]
				podlist, err = utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
				Expect(err).To(BeNil())
				StatusCode := utils.DeleteDeployment(ctx.Cfg.ApiServer+DeploymentHandler, deployment.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.ApiServer+AppHandler, podlist)
			utils.PrintTestcaseNameandStatus()
		})

		FIt("E2E_APP_DEPLOYMENT_1: Test application deployment in edge_core", func() {
			var deploymentList v1.DeploymentList
			var podlist metav1.PodList
			replica := 1
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandleDeployment(http.MethodPost, ctx.Cfg.ApiServer+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], replica)
			Expect(IsAppDeployed).Should(BeTrue())
			//CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, UID)
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.ApiServer+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := deployment.ObjectMeta.Labels["app"]
					podlist, err = utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
		})
	})
})
