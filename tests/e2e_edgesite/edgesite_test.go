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

package edgesite

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/testsuite"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var DeploymentTestTimerGroup = utils.NewTestTimerGroup()

// Run Test cases
var _ = Describe("Application deployment test in E2E scenario using EdgeSite", func() {
	var UID string
	var testTimer *utils.TestTimer
	var testSpecReport SpecReport

	var clientSet clientset.Interface

	BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
	})

	Context("Test application deployment and delete deployment using deployment spec", func() {
		BeforeEach(func() {
			// Get current test SpecReport
			testSpecReport = CurrentSpecReport()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})

		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()

			By(fmt.Sprintf("get deployment %s", UID))
			deployment, err := utils.GetDeployment(clientSet, metav1.NamespaceDefault, UID)
			Expect(err).To(BeNil())

			By(fmt.Sprintf("list pod for deploy %s", UID))
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			_, err = utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			Expect(err).To(BeNil())

			By(fmt.Sprintf("delete deploy %s", UID))
			err = utils.DeleteDeployment(clientSet, deployment.Namespace, deployment.Name)
			Expect(err).To(BeNil())

			By(fmt.Sprintf("wait for pod of deploy %s to disappear", UID))
			err = utils.WaitForPodsToDisappear(clientSet, metav1.NamespaceDefault, labelSelector, constants.Interval, constants.Timeout)
			Expect(err).To(BeNil())

			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_ES_APP_DEPLOYMENT_1: Create deployment and check the pods are coming up correctly", func() {
			replica := int32(1)
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			testsuite.CreateDeploymentTest(clientSet, replica, UID)
		})

		It("E2E_ES_APP_DEPLOYMENT_2: Create deployment with replicas and check the pods are coming up correctly", func() {
			replica := int32(3)
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			testsuite.CreateDeploymentTest(clientSet, replica, UID)
		})

		It("E2E_ES_APP_DEPLOYMENT_3: Create deployment and check deployment ctrler re-creating pods when user deletes the pods manually", func() {
			replica := int32(3)
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			podList := testsuite.CreateDeploymentTest(clientSet, replica, UID)
			for _, pod := range podList.Items {
				err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				Expect(err).To(BeNil())
			}
			utils.CheckPodDeleteState(clientSet, podList)

			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			podList, err := utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			Expect(err).To(BeNil())
			Expect(len(podList.Items)).Should(Equal(replica))

			utils.WaitForPodsRunning(clientSet, podList, 240*time.Second)
		})

	})
	Context("Test application deployment using Pod spec using EdgeSite", func() {
		BeforeEach(func() {
			// Get current test SpecReport
			testSpecReport = CurrentSpecReport()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})

		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()

			labelSelector := labels.SelectorFromSet(constants.KubeEdgeE2ELabel)
			podList, err := utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			Expect(err).To(BeNil())

			for _, pod := range podList.Items {
				err = utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				Expect(err).To(BeNil())
			}

			utils.CheckPodDeleteState(clientSet, podList)

			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_ES_POD_DEPLOYMENT_1: Create a pod and check the pod is coming up correctly", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPod(podName, ctx.Cfg.AppImageURL[0])

			testsuite.CreatePodTest(clientSet, pod)
		})

		It("E2E_ES_POD_DEPLOYMENT_2: Create the pod and delete pod happening successfully", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPod(podName, ctx.Cfg.AppImageURL[0])

			podList := testsuite.CreatePodTest(clientSet, pod)
			for _, pod := range podList.Items {
				err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				Expect(err).To(BeNil())
			}
			utils.CheckPodDeleteState(clientSet, podList)
		})

		It("E2E_ES_POD_DEPLOYMENT_3: Create pod and delete the pod successfully, and delete already deleted pod and check the behaviour", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPod(podName, ctx.Cfg.AppImageURL[0])

			podList := testsuite.CreatePodTest(clientSet, pod)
			for _, pod := range podList.Items {
				err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				Expect(err).To(BeNil())
			}
			utils.CheckPodDeleteState(clientSet, podList)

			err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

		It("E2E_ES_POD_DEPLOYMENT_4: Create and delete pod multiple times and check all the Pod created and deleted successfully", func() {
			//Generate the random string and assign as a UID
			for i := 0; i < 10; i++ {
				//Generate the random string and assign as podName
				podName := "pod-app-" + utils.GetRandomString(5)
				pod := utils.NewPod(podName, ctx.Cfg.AppImageURL[0])

				podList := testsuite.CreatePodTest(clientSet, pod)
				for _, pod := range podList.Items {
					err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
					Expect(err).To(BeNil())
				}
				utils.CheckPodDeleteState(clientSet, podList)
			}
		})
	})
})
