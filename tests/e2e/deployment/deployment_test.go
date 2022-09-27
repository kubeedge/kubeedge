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
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	. "github.com/kubeedge/kubeedge/tests/e2e/testsuite"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var DeploymentTestTimerGroup = utils.NewTestTimerGroup()

//Run Test cases
var _ = Describe("Application deployment test in E2E scenario", func() {
	var UID string
	var testTimer *utils.TestTimer
	var testSpecReport SpecReport

	var clientSet clientset.Interface

	BeforeEach(func() {
		clientSet = utils.NewKubeClient(ctx.Cfg.KubeConfigPath)
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
			var podlist corev1.PodList
			var deploymentList appsv1.DeploymentList
			err := utils.GetDeployments(&deploymentList, ctx.Cfg.K8SMasterForKubeEdge+constants.DeploymentHandler)
			Expect(err).To(BeNil())
			for _, deployment := range deploymentList.Items {
				if deployment.Name == UID {
					label := nodeName
					podlist, err = utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, label)
					Expect(err).To(BeNil())
					StatusCode := utils.DeleteDeployment(ctx.Cfg.K8SMasterForKubeEdge+constants.DeploymentHandler, deployment.Name)
					Expect(StatusCode).Should(Equal(http.StatusOK))
				}
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_APP_DEPLOYMENT_1: Create deployment and check the pods are coming up correctly", func() {
			replica := 1
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			CreateDeploymentTest(replica, UID, nodeName, nodeSelector, ctx)
		})
		It("E2E_APP_DEPLOYMENT_2: Create deployment with replicas and check the pods are coming up correctly", func() {
			replica := 3
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			CreateDeploymentTest(replica, UID, nodeName, nodeSelector, ctx)
		})

		It("E2E_APP_DEPLOYMENT_3: Create deployment and check deployment ctrler re-creating pods when user deletes the pods manually", func() {
			replica := 3
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			podlist := CreateDeploymentTest(replica, UID, nodeName, nodeSelector, ctx)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, label)
			Expect(err).To(BeNil())
			Expect(len(podlist.Items)).Should(Equal(replica))
			utils.WaitforPodsRunning(ctx.Cfg.KubeConfigPath, podlist, 240*time.Second)
		})

	})
	Context("Test application deployment using Pod spec", func() {
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
			var podlist corev1.PodList
			label := nodeName
			podlist, err := utils.GetPods(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, label)
			Expect(err).To(BeNil())
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
			utils.PrintTestcaseNameandStatus()
		})

		It("E2E_POD_DEPLOYMENT_1: Create a pod and check the pod is coming up correctly", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPodObj(podName, ctx.Cfg.AppImageURL[0], nodeSelector)

			CreatePodTest(nodeName, podName, ctx, pod)
		})

		It("E2E_POD_DEPLOYMENT_2: Create the pod and delete pod happening successfully", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPodObj(podName, ctx.Cfg.AppImageURL[0], nodeSelector)

			podlist := CreatePodTest(nodeName, podName, ctx, pod)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
		})
		It("E2E_POD_DEPLOYMENT_3: Create pod and delete the pod successfully, and delete already deleted pod and check the behaviour", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPodObj(podName, ctx.Cfg.AppImageURL[0], nodeSelector)

			podlist := CreatePodTest(nodeName, podName, ctx, pod)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
			_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler + "/" + UID)
			Expect(StatusCode).Should(Equal(http.StatusNotFound))
		})
		It("E2E_POD_DEPLOYMENT_4: Create and delete pod multiple times and check all the Pod created and deleted successfully", func() {
			//Generate the random string and assign as a UID
			for i := 0; i < 10; i++ {
				//Generate the random string and assign as podName
				podName := "pod-app-" + utils.GetRandomString(5)
				pod := utils.NewPodObj(podName, ctx.Cfg.AppImageURL[0], nodeSelector)

				podlist := CreatePodTest(nodeName, podName, ctx, pod)
				for _, pod := range podlist.Items {
					_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler + "/" + pod.Name)
					Expect(StatusCode).Should(Equal(http.StatusOK))
				}
				utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
			}
		})
		It("E2E_POD_DEPLOYMENT_5: Create pod with hostpath volume successfully", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPodObj(podName, ctx.Cfg.AppImageURL[0], nodeSelector)

			pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
				Name:      "hp",
				MountPath: "/hp",
			}}
			pod.Spec.Volumes = []corev1.Volume{{
				Name: "hp",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/tmp"},
				},
			}}

			podlist := CreatePodTest(nodeName, podName, ctx, pod)
			for _, pod := range podlist.Items {
				_, StatusCode := utils.DeletePods(ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler + "/" + pod.Name)
				Expect(StatusCode).Should(Equal(http.StatusOK))
			}
			utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)
		})
	})

	Context("StatefulSet lifecycle test in edge node", func() {
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

			By(fmt.Sprintf("get StatefulSet %s", UID))
			statefulSet, err := utils.GetStatefulSet(clientSet, metav1.NamespaceDefault, UID)
			Expect(err).To(BeNil())

			By(fmt.Sprintf("list pod for StatefulSet %s", UID))
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			_, err = utils.ListPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			Expect(err).To(BeNil())

			By(fmt.Sprintf("delete StatefulSet %s", UID))
			err = utils.DeleteStatefulSet(clientSet, statefulSet.Namespace, statefulSet.Name)
			Expect(err).To(BeNil())

			By(fmt.Sprintf("wait for pod of StatefulSet %s disappear", UID))
			err = utils.WaitForPodsToDisappear(clientSet, metav1.NamespaceDefault, labelSelector, constants.Interval, constants.Timeout)
			Expect(err).To(BeNil())

			utils.PrintTestcaseNameandStatus()
		})

		It("Basic StatefulSet test", func() {
			replica := int32(2)
			// Generate the random string and assign as a UID
			UID = "edge-statefulset-" + utils.GetRandomString(5)

			By(fmt.Sprintf("create StatefulSet %s", UID))
			d := utils.NewTestStatefulSet(UID, ctx.Cfg.AppImageURL[1], replica)
			ss, err := utils.CreateStatefulSet(clientSet, d)
			Expect(err).To(BeNil())

			utils.WaitForStatusReplicas(clientSet, ss, replica)

			By(fmt.Sprintf("get pod for StatefulSet %s", UID))
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			podList, err := utils.ListPods(clientSet, corev1.NamespaceDefault, labelSelector, nil)
			Expect(err).To(BeNil())
			Expect(len(podList.Items)).ShouldNot(Equal(0))

			By(fmt.Sprintf("wait for pod of StatefulSet %s running", UID))
			utils.WaitforPodsRunning(ctx.Cfg.KubeConfigPath, *podList, 240*time.Second)
		})
		It("Delete statefulSet pod multi times", func() {
			replica := int32(2)
			// Generate the random string and assign as a UID
			UID = "edge-statefulset-" + utils.GetRandomString(5)

			By(fmt.Sprintf("create StatefulSet %s", UID))
			d := utils.NewTestStatefulSet(UID, ctx.Cfg.AppImageURL[1], replica)
			ss, err := utils.CreateStatefulSet(clientSet, d)
			Expect(err).To(BeNil())

			utils.WaitForStatusReplicas(clientSet, ss, replica)

			By(fmt.Sprintf("get pod for StatefulSet %s", UID))
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			podList, err := utils.ListPods(clientSet, corev1.NamespaceDefault, labelSelector, nil)
			Expect(err).To(BeNil())
			Expect(len(podList.Items)).ShouldNot(Equal(0))

			By(fmt.Sprintf("wait for pod of StatefulSet %s running", UID))
			utils.WaitforPodsRunning(ctx.Cfg.KubeConfigPath, *podList, 240*time.Second)

			deletePodName := fmt.Sprintf("%s-1", UID)
			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("delete pod %s", deletePodName))
				err = utils.DeletePod(clientSet, deletePodName, "default")
				Expect(err).To(BeNil())

				By(fmt.Sprintf("wait for pod %s running again", fmt.Sprintf("%s-1", UID)))
				err = wait.Poll(5*time.Second, 120*time.Second, func() (bool, error) {
					pod, err := clientSet.CoreV1().Pods("default").Get(context.TODO(), deletePodName, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					if pod.Status.Phase == corev1.PodRunning {
						return true, nil
					}
					return false, nil
				})
				Expect(err).To(BeNil())
			}
		})
	})
})
