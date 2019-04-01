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
var _ = Describe("Application deployment test in E2E scenario", func() {
	var UID string
	Context("Test application deployment and delete deployment using deployment spec", func() {
		BeforeEach(func() {
		})
		AfterEach(func() {
			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.ApiServer+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
					Expect(err).To(BeNil())
					StatusCode := utils.DeleteDeployment(ctx.Cfg.ApiServer+DeploymentHandler, deployment.Name)
					Expect(StatusCode).Should(Equal(http.StatusOK))
				}
			}
			utils.CheckPodDeleteState(ctx.Cfg.ApiServer+AppHandler, podlist)
			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_APP_DEPLOYMENT_1: Create deployment and check the pods are coming up correctly", func() {
			var deploymentList v1.DeploymentList
			var podlist metav1.PodList
			replica := 1
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandleDeployment(http.MethodPost, ctx.Cfg.ApiServer+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], nodeSelector, replica)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.ApiServer+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
		})
		It("E2E_APP_DEPLOYMENT_2: Create deployment with replicas and check the pods are coming up correctly", func() {
			var deploymentList v1.DeploymentList
			var podlist metav1.PodList
			replica := 3
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandleDeployment(http.MethodPost, ctx.Cfg.ApiServer+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], nodeSelector, replica)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.ApiServer+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
		})

		It("E2E_APP_DEPLOYMENT_3: Create deployment and check deployment ctrler re-creating pods when user deletes the pods manually", func() {
			var deploymentList v1.DeploymentList
			var podlist metav1.PodList
			replica := 3
			//Generate the random string and assign as a UID
			UID = "deployment-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandleDeployment(http.MethodPost, ctx.Cfg.ApiServer+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], nodeSelector, replica)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.ApiServer+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.ApiServer+AppHandler+"/"+pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.ApiServer+AppHandler, podlist)
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			Expect(len(podlist.Items)).Should(Equal(replica))
			utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
		})

	})
	Context("Test application deployment using Pod spec", func() {
		BeforeEach(func() {
		})
		AfterEach(func() {
			var podlist metav1.PodList
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
			Expect(err).To(BeNil())
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.ApiServer+AppHandler+"/"+pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.ApiServer+AppHandler, podlist)
			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_POD_DEPLOYMENT_1: Create a pod and check the pod is coming up correclty", func() {
			var podlist metav1.PodList
			//Generate the random string and assign as a UID
			UID = "pod-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.ApiServer+AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
			Expect(IsAppDeployed).Should(BeTrue())
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
			Expect(err).To(BeNil())
			utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
		})

		It("E2E_POD_DEPLOYMENT_2: Create the pod and delete pod happening successfully", func() {
			var podlist metav1.PodList
			//Generate the random string and assign as a UID
			UID = "pod-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.ApiServer+AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
			Expect(IsAppDeployed).Should(BeTrue())
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
			Expect(err).To(BeNil())
			utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.ApiServer+AppHandler+"/"+pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.ApiServer+AppHandler, podlist)

		})
		It("E2E_POD_DEPLOYMENT_3: Create pod and delete the pod successfully, and delete already deleted pod and check the behaviour", func() {
			var podlist metav1.PodList
			//Generate the random string and assign as a UID
			UID = "pod-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.ApiServer+AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
			Expect(IsAppDeployed).Should(BeTrue())
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
			Expect(err).To(BeNil())
			utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.ApiServer+AppHandler+"/"+pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.ApiServer+AppHandler, podlist)
			_, StatusCode := utils.DeletePods(ctx.Cfg.ApiServer+AppHandler+"/"+UID)
			Expect(StatusCode).Should(Equal(http.StatusNotFound))

		})
		It("E2E_POD_DEPLOYMENT_4: Create and delete pod multiple times and check all the Pod created and deleted successfully", func() {
			//Generate the random string and assign as a UID
			for i:=0; i<10; i++{
				UID = "pod-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.ApiServer+AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
				Expect(IsAppDeployed).Should(BeTrue())
				label := nodeName
				podlist, err := utils.GetPods(ctx.Cfg.ApiServer+AppHandler, label)
				Expect(err).To(BeNil())
				utils.CheckPodRunningState(ctx.Cfg.ApiServer+AppHandler, podlist)
				for _, pod := range podlist.Items {
					_, StatusCode := utils.DeletePods(ctx.Cfg.ApiServer+AppHandler+"/"+pod.Name)
					Expect(StatusCode).Should(Equal(http.StatusOK))
				}
				utils.CheckPodDeleteState(ctx.Cfg.ApiServer+AppHandler, podlist)
			}
		})
	})
})