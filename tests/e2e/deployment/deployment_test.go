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
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	AppHandler        = "/api/v1/namespaces/default/pods"
	NodeHandler       = "/api/v1/nodes"
	DeploymentHandler = "/apis/apps/v1/namespaces/default/deployments"
	ServiceHandler    = "/api/v1/namespaces/default/services"
)

var DeploymentTestTimerGroup *utils.TestTimerGroup = utils.NewTestTimerGroup()

//Run Test cases
var _ = Describe("Application deployment test in E2E scenario", func() {
	var UID string
	var testTimer *utils.TestTimer
	var testDescription GinkgoTestDescription
	Context("Test application deployment and delete deployment using deployment spec", func() {
		BeforeEach(func() {
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
					Expect(err).To(BeNil())
					StatusCode := utils.DeleteDeployment(ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, deployment.Name)
					Expect(StatusCode).Should(Equal(http.StatusOK))
				}
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_APP_DEPLOYMENT_1: Create deployment and check the pods are coming up correctly", func() {
			var deploymentList v1.DeploymentList
			var podlist metav1.PodList
			replica := 1
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], nodeSelector, "", replica)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
		})
		It("E2E_APP_DEPLOYMENT_2: Create deployment with replicas and check the pods are coming up correctly", func() {
			var deploymentList v1.DeploymentList
			var podlist metav1.PodList
			replica := 3
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], nodeSelector, "", replica)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
		})

		It("E2E_APP_DEPLOYMENT_3: Create deployment and check deployment ctrler re-creating pods when user deletes the pods manually", func() {
			var deploymentList v1.DeploymentList
			var podlist metav1.PodList
			replica := 3
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[1], nodeSelector, "", replica)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
					Expect(err).To(BeNil())
					break
				}
			}
			Expect(len(podlist.Items)).Should(Equal(replica))
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
		})

	})
	Context("Test application deployment using Pod spec", func() {
		BeforeEach(func() {
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			var podlist metav1.PodList
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
			Expect(err).To(BeNil())
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_POD_DEPLOYMENT_1: Create a pod and check the pod is coming up correclty", func() {
			var podlist metav1.PodList
			//Generate the random string and assign as a UID
			UID = "pod-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
			Expect(IsAppDeployed).Should(BeTrue())
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
			Expect(err).To(BeNil())
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
		})

		It("E2E_POD_DEPLOYMENT_2: Create the pod and delete pod happening successfully", func() {
			var podlist metav1.PodList
			//Generate the random string and assign as a UID
			UID = "pod-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
			Expect(IsAppDeployed).Should(BeTrue())
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
			Expect(err).To(BeNil())
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
		})
		It("E2E_POD_DEPLOYMENT_3: Create pod and delete the pod successfully, and delete already deleted pod and check the behaviour", func() {
			var podlist metav1.PodList
			//Generate the random string and assign as a UID
			UID = "pod-app-" + utils.GetRandomString(5)
			IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
			Expect(IsAppDeployed).Should(BeTrue())
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
			Expect(err).To(BeNil())
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + UID)
			Expect(StatusCode).Should(Equal(http.StatusNotFound))
		})
		It("E2E_POD_DEPLOYMENT_4: Create and delete pod multiple times and check all the Pod created and deleted successfully", func() {
			//Generate the random string and assign as a UID
			for i := 0; i < 10; i++ {
				UID = "pod-app-" + utils.GetRandomString(5)
				IsAppDeployed := utils.HandlePod(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+AppHandler, UID, ctx.Cfg.AppImageUrl[0], nodeSelector)
				Expect(IsAppDeployed).Should(BeTrue())
				label := nodeName
				podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
				Expect(err).To(BeNil())
				utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)
				for _, pod := range podlist.Items {
					_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + pod.Name)
					Expect(StatusCode).Should(Equal(http.StatusOK))
				}
				utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			}
		})
	})
	Context("Test Pod communiction with edgeMesh", func() {
		BeforeEach(func() {
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				label := nodeName
				podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
				Expect(err).To(BeNil())
				StatusCode := utils.DeleteDeployment(ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, deployment.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)

			var serviceList metav1.ServiceList
			err = utils.GetServices(&serviceList, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler+utils.LabelSelector+"service%3Dtest")
			Expect(err).To(BeNil())
			for _, service := range serviceList.Items {
				StatusCode := utils.DeleteSvc(ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler+"/"+ service.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_SERVICE_EDGEMESH_1: Create two pods and check the pods are communicating or not", func() {
			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			var servicelist metav1.ServiceList
			//Generate the random string and assign as a UID
			UID = "pod-app-server"
			IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 30)
			for _, deployment := range deploymentList.Items {
				utils.Info("delpyment label : %+v", deployment.Labels)
				label := nodeName
				podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
				Expect(err).To(BeNil())
				utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
			}
			utils.Info("\n Server app deployed \n")
			// time.Sleep(time.Second * 30)

			// Deploy service over the server pod
			err = utils.ExposePodService(UID, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler, 80, intstr.FromInt(8000))
			Expect(err).To(BeNil())
			err = utils.GetServices(&servicelist, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
			Expect(err).To(BeNil())

			// Check server app is accessible with default value
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))

			UID = "pod-app-client"
			IsAppDeployed = utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[3], nodeSelector, "", 1)
			Expect(IsAppDeployed).Should(BeTrue())
			err = utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())

			// Check weather the name variable is changed in server
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Changed"))

			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, label)
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
					break
				}
			}
		})
		It("E2E_SERVICE_EDGEMESH_2: Client pod restart: POSITIVE", func() {
			//deploy server deployment
			//check pods running
			//deploy service
			//

			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			var servicelist metav1.ServiceList
			var deployment v1.Deployment
			//Generate the random string and assign as a UID
			UID = "pod-app-server"
			//IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			depobj := utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[2], nodeSelector, 1, map[string]string{"app": "server"}, 80)
			IsAppDeployed := utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 30)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Server app deployed \n")
			// time.Sleep(time.Second * 30)

			// Deploy service over the server pod
			err = utils.ExposePodService(UID, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler, 80, intstr.FromInt(8000))
			Expect(err).To(BeNil())
			err = utils.GetServices(&servicelist, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
			Expect(err).To(BeNil())

			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))

			//deploy client deployment
			UID = "pod-app-client"
			//IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			depobj = utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[3], nodeSelector, 1, map[string]string{"app": "client"}, 81)
			IsAppDeployed = utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err = utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 20)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Client app deployed \n")

			//check name changed(communiction happened)
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))

			//delete client
			err = utils.GetDeployment(&deployment, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler,UID)
			Expect(err).To(BeNil())

			label := deployment.Spec.Selector
			for key, value := range label.MatchLabels {
				podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
				Expect(err).To(BeNil())
				for _, pod := range podlist.Items {
					_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + pod.Name)
					Expect(StatusCode).Should(Equal(http.StatusOK))
				}
			}

			//_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + UID)
			//Expect(StatusCode).Should(Equal(http.StatusOK))

			//change the name back to default again
			var jsonStr = []byte("Changed")
			resp, err := http.Post("http://localhost:8000/hello", "application/json", bytes.NewBuffer(jsonStr))
			//req, _ := http.NewRequest("POST", "http://localhost:8000", bytes.NewBuffer(jsonStr))
			//req.Header.Set("Content-Type", "application/json")
			//client := &http.Client{}
			//resp, err := client.Do(req)
			if err != nil {
				panic(err)
			} else {
				utils.Info("---%v", resp)
				content, _ := ioutil.ReadAll(resp.Body)
				utils.Info("body, %s", content)
			}

			//deployment will restart it check again pod is there
			podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+"app"+"%3D"+"client", "")
			Expect(err).To(BeNil())
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)

			//check the name is changed of not
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Changed"))

		})
		It("E2E_SERVICE_EDGEMESH_3: Server pod restart: POSITIVE", func() {
			//deploy server deployment
			//check pods running
			//deploy service

			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			var servicelist metav1.ServiceList
			var deployment v1.Deployment
			//Generate the random string and assign as a UID
			UID = "pod-app-server"
			//IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			depobj := utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[2], nodeSelector, 1, map[string]string{"app": "server"}, 80)
			IsAppDeployed := utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 30)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Server app deployed \n")
			// time.Sleep(time.Second * 30)

			// Deploy service over the server pod
			err = utils.ExposePodService(UID, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler, 80, intstr.FromInt(8000))
			Expect(err).To(BeNil())
			err = utils.GetServices(&servicelist, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
			Expect(err).To(BeNil())

			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))

			//deploy client deployment
			UID = "pod-app-client"
			//IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			depobj = utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[3], nodeSelector, 1, map[string]string{"app": "client"}, 81)
			IsAppDeployed = utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err = utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 20)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Client app deployed \n")

			//check name changed(communiction happened)
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))

			//delete server pod
			err = utils.GetDeployment(&deployment, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler,"pod-app-server")
			Expect(err).To(BeNil())

			label := deployment.Spec.Selector
			for key, value := range label.MatchLabels {
				podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
				Expect(err).To(BeNil())
				for _, pod := range podlist.Items {
					_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + AppHandler + "/" + pod.Name)
					Expect(StatusCode).Should(Equal(http.StatusOK))
				}
			}

			//deployment will restart it check again pod is there
			podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+"app"+"%3D"+"server", "")
			Expect(err).To(BeNil())
			utils.WaitforPodsRunning(ctx.Cfg.K8SMasterForKubeEdge, podlist, 240*time.Second)

			//check the name is changed of not
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Changed"))
		})
		It("E2E_SERVICE_EDGEMESH_4: Server deployment gets deleted: FAILURE", func() {
			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			var servicelist metav1.ServiceList
			//Generate the random string and assign as a UID
			UID = "pod-app-server"
			//IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			depobj := utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[2], nodeSelector, 1, map[string]string{"app": "server"}, 80)
			IsAppDeployed := utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 30)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Server app deployed \n")
			// time.Sleep(time.Second * 30)

			// Deploy service over the server pod
			err = utils.ExposePodService(UID, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler, 80, intstr.FromInt(8000))
			Expect(err).To(BeNil())
			err = utils.GetServices(&servicelist, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
			Expect(err).To(BeNil())

			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))

			//deploy client deployment
			UID = "pod-app-client"
			//IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			depobj = utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[3], nodeSelector, 1, map[string]string{"app": "client"}, 81)
			IsAppDeployed = utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err = utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 20)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Client app deployed \n")

			//check name changed(communiction happened)
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Changed"))


			//delete server deployment
			UID = "pod-app-server"
			StatusCode := utils.DeleteDeployment(ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID)
			Expect(StatusCode).Should(Equal(http.StatusOK))

			//deploy again with same deployment name
			depobj = utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[2], nodeSelector, 1, map[string]string{"app": "server"}, 80)
			IsAppDeployed = utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err = utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 30)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Server app deployed \n")

			//check the name is should not have been changed
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))
		})
		FIt("E2E_SERVICE_EDGEMESH_5: delete service : FAILURE", func() {
			var podlist metav1.PodList
			var deploymentList v1.DeploymentList
			var servicelist metav1.ServiceList
			//Generate the random string and assign as a UID
			UID = "pod-app-server"
			//IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			depobj := utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[2], nodeSelector, 1, map[string]string{"app": "server"}, 80)
			IsAppDeployed := utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 30)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Server app deployed \n")
			// time.Sleep(time.Second * 30)

			// Deploy service over the server pod
			err = utils.ExposePodService(UID, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler, 80, intstr.FromInt(8000))
			Expect(err).To(BeNil())
			err = utils.GetServices(&servicelist, ctx.Cfg.K8SMasterForKubeEdge+ServiceHandler)
			Expect(err).To(BeNil())

			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))

			//deploy client deployment
			UID = "pod-app-client"
			//IsAppDeployed := utils.HandleDeployment(false, false, http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, ctx.Cfg.AppImageUrl[2], nodeSelector, "", 1)
			depobj = utils.CreateDeployment(UID, ctx.Cfg.AppImageUrl[3], nodeSelector, 1, map[string]string{"app": "client"}, 81)
			IsAppDeployed = utils.HandleRequest(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler, UID, depobj)
			Expect(IsAppDeployed).Should(BeTrue())
			err = utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+DeploymentHandler+utils.LabelSelector+"app%3Dkubeedge")
			Expect(err).To(BeNil())
			time.Sleep(time.Second * 20)
			for _, deployment := range deploymentList.Items {
				label := deployment.Spec.Selector
				for key, value := range label.MatchLabels {
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+AppHandler+utils.LabelSelector+key+"%3D"+value, "")
					Expect(err).To(BeNil())
					utils.CheckPodRunningState(ctx.Cfg.K8SMasterForKubeEdge+AppHandler, podlist)
				}
			}
			utils.Info("\n Client app deployed \n")

			//check name changed(communiction happened)
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))

			//delete service
			StatusCode := utils.DeleteSvc(ctx.Cfg.K8SMasterForKubeEdge + ServiceHandler + "/" + "pod-app-server")
			Expect(StatusCode).Should(Equal(http.StatusOK))

			//change the name back to default again
			var jsonStr = []byte("Changed")
			resp, err := http.Post("http://localhost:8000/hello", "application/json", bytes.NewBuffer(jsonStr))
			if err != nil {
				panic(err)
			} else {
				content, _ := ioutil.ReadAll(resp.Body)
				utils.Info("body, %s", content)
			}

			//check the name should not have been changed
			Expect(utils.Getname("http://localhost:8000")).To(BeEquivalentTo("Default"))
		})
		It("E2E_SERVICE_EDGEMESH_6: create Loadbalancer service : FAILURE", func() {
			//deploy server deployment
			//check pods running
			//deploy service with loadbalancer
			//
			//deploy client deployment
			//
			//check the name should not have changed
		})
	})
})
