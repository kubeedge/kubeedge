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

package apps

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/testsuite"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var DeploymentTestTimerGroup = utils.NewTestTimerGroup()

// Run Test cases
var _ = GroupDescribe("Application deployment test in E2E scenario", func() {
	var UID string
	var testTimer *utils.TestTimer
	var testSpecReport ginkgo.GinkgoTestDescription

	var clientSet clientset.Interface

	ginkgo.BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
	})

	ginkgo.Context("Test application deployment and delete deployment using deployment spec", func() {
		ginkgo.BeforeEach(func() {
			// Get current test SpecReport
			testSpecReport = ginkgo.CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testSpecReport.TestText)
		})

		ginkgo.AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()

			ginkgo.By(fmt.Sprintf("get deployment %s", UID))
			deployment, err := utils.GetDeployment(clientSet, metav1.NamespaceDefault, UID)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("list pod for deploy %s", UID))
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			_, err = utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("delete deploy %s", UID))
			err = utils.DeleteDeployment(clientSet, deployment.Namespace, deployment.Name)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("wait for pod of deploy %s to disappear", UID))
			err = utils.WaitForPodsToDisappear(clientSet, metav1.NamespaceDefault, labelSelector, constants.Interval, constants.Timeout)
			gomega.Expect(err).To(gomega.BeNil())

			utils.PrintTestcaseNameandStatus()
		})

		ginkgo.It("E2E_APP_DEPLOYMENT_1: Create deployment and check the pods are coming up correctly", func() {
			replica := int32(1)
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			testsuite.CreateDeploymentTest(clientSet, replica, UID)
		})

		ginkgo.It("E2E_APP_DEPLOYMENT_2: Create deployment with replicas and check the pods are coming up correctly", func() {
			replica := int32(3)
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			testsuite.CreateDeploymentTest(clientSet, replica, UID)
		})

		ginkgo.It("E2E_APP_DEPLOYMENT_3: Create deployment and check deployment ctrler re-creating pods when user deletes the pods manually", func() {
			replica := int32(3)
			//Generate the random string and assign as a UID
			UID = "edgecore-depl-app-" + utils.GetRandomString(5)
			podList := testsuite.CreateDeploymentTest(clientSet, replica, UID)
			for _, pod := range podList.Items {
				err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				gomega.Expect(err).To(gomega.BeNil())
			}
			utils.CheckPodDeleteState(clientSet, podList)

			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			podList, err := utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(podList.Items)).Should(gomega.Equal(int(replica)))

			utils.WaitForPodsRunning(clientSet, podList, 240*time.Second)
		})

	})

	ginkgo.Context("Test application deployment using Pod spec", func() {
		ginkgo.BeforeEach(func() {
			// Get current test SpecReport
			testSpecReport = ginkgo.CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testSpecReport.TestText)
		})
		ginkgo.AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()

			labelSelector := labels.SelectorFromSet(constants.KubeEdgeE2ELabel)
			podList, err := utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())

			for _, pod := range podList.Items {
				err = utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				gomega.Expect(err).To(gomega.BeNil())
			}

			utils.CheckPodDeleteState(clientSet, podList)

			utils.PrintTestcaseNameandStatus()
		})

		ginkgo.It("E2E_POD_DEPLOYMENT_1: Create a pod and check the pod is coming up correctly", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPod(podName, utils.LoadConfig().AppImageURL[0])

			testsuite.CreatePodTest(clientSet, pod)
		})

		ginkgo.It("E2E_POD_DEPLOYMENT_2: Create the pod and delete pod happening successfully", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPod(podName, utils.LoadConfig().AppImageURL[0])

			podList := testsuite.CreatePodTest(clientSet, pod)
			for _, pod := range podList.Items {
				err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				gomega.Expect(err).To(gomega.BeNil())
			}

			utils.CheckPodDeleteState(clientSet, podList)
		})

		ginkgo.It("E2E_POD_DEPLOYMENT_3: Create pod and delete the pod successfully, and delete already deleted pod and check the behaviour", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPod(podName, utils.LoadConfig().AppImageURL[0])

			podList := testsuite.CreatePodTest(clientSet, pod)
			for _, pod := range podList.Items {
				err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				gomega.Expect(err).To(gomega.BeNil())
			}

			utils.CheckPodDeleteState(clientSet, podList)

			err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
			gomega.Expect(errors.IsNotFound(err)).To(gomega.BeTrue())
		})

		ginkgo.It("E2E_POD_DEPLOYMENT_4: Create and delete pod multiple times and check all the Pod created and deleted successfully", func() {
			//Generate the random string and assign as a UID
			for i := 0; i < 10; i++ {
				//Generate the random string and assign as podName
				podName := "pod-app-" + utils.GetRandomString(5)
				pod := utils.NewPod(podName, utils.LoadConfig().AppImageURL[0])

				podList := testsuite.CreatePodTest(clientSet, pod)
				for _, pod := range podList.Items {
					err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
					gomega.Expect(err).To(gomega.BeNil())
				}
				utils.CheckPodDeleteState(clientSet, podList)
			}
		})

		ginkgo.It("E2E_POD_DEPLOYMENT_5: Create pod with hostpath volume successfully", func() {
			//Generate the random string and assign as podName
			podName := "pod-app-" + utils.GetRandomString(5)
			pod := utils.NewPod(podName, utils.LoadConfig().AppImageURL[0])

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

			podList := testsuite.CreatePodTest(clientSet, pod)
			for _, pod := range podList.Items {
				err := utils.DeletePod(clientSet, pod.Namespace, pod.Name)
				gomega.Expect(err).To(gomega.BeNil())
			}
			utils.CheckPodDeleteState(clientSet, podList)
		})
	})

	ginkgo.Context("StatefulSet lifecycle test in edge node", func() {
		ginkgo.BeforeEach(func() {
			// Get current test SpecReport
			testSpecReport = ginkgo.CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = DeploymentTestTimerGroup.NewTestTimer(testSpecReport.TestText)
		})

		ginkgo.AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()

			ginkgo.By(fmt.Sprintf("get StatefulSet %s", UID))
			statefulSet, err := utils.GetStatefulSet(clientSet, metav1.NamespaceDefault, UID)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("list pod for StatefulSet %s", UID))
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			_, err = utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("delete StatefulSet %s", UID))
			err = utils.DeleteStatefulSet(clientSet, statefulSet.Namespace, statefulSet.Name)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("wait for pod of StatefulSet %s disappear", UID))
			err = utils.WaitForPodsToDisappear(clientSet, metav1.NamespaceDefault, labelSelector, constants.Interval, constants.Timeout)
			gomega.Expect(err).To(gomega.BeNil())

			utils.PrintTestcaseNameandStatus()
		})

		ginkgo.It("Basic StatefulSet test", func() {
			replica := int32(2)
			// Generate the random string and assign as a UID
			UID = "edge-statefulset-" + utils.GetRandomString(5)

			ginkgo.By(fmt.Sprintf("create StatefulSet %s", UID))
			d := utils.NewTestStatefulSet(UID, utils.LoadConfig().AppImageURL[1], replica)
			ss, err := utils.CreateStatefulSet(clientSet, d)
			gomega.Expect(err).To(gomega.BeNil())

			utils.WaitForStatusReplicas(clientSet, ss, replica)

			ginkgo.By(fmt.Sprintf("get pod for StatefulSet %s", UID))
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			podList, err := utils.GetPods(clientSet, corev1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(podList.Items)).ShouldNot(gomega.Equal(0))

			ginkgo.By(fmt.Sprintf("wait for pod of StatefulSet %s running", UID))
			utils.WaitForPodsRunning(clientSet, podList, 240*time.Second)
		})
		ginkgo.It("Delete statefulSet pod multi times", func() {
			replica := int32(2)
			// Generate the random string and assign as a UID
			UID = "edge-statefulset-" + utils.GetRandomString(5)

			ginkgo.By(fmt.Sprintf("create StatefulSet %s", UID))
			d := utils.NewTestStatefulSet(UID, utils.LoadConfig().AppImageURL[1], replica)
			ss, err := utils.CreateStatefulSet(clientSet, d)
			gomega.Expect(err).To(gomega.BeNil())

			utils.WaitForStatusReplicas(clientSet, ss, replica)

			ginkgo.By(fmt.Sprintf("get pod for StatefulSet %s", UID))
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			podList, err := utils.GetPods(clientSet, corev1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(podList.Items)).ShouldNot(gomega.Equal(0))

			ginkgo.By(fmt.Sprintf("wait for pod of StatefulSet %s running", UID))
			utils.WaitForPodsRunning(clientSet, podList, 240*time.Second)

			deletePodName := fmt.Sprintf("%s-1", UID)
			for i := 0; i < 5; i++ {
				ginkgo.By(fmt.Sprintf("delete pod %s", deletePodName))
				err = utils.DeletePod(clientSet, "default", deletePodName)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By(fmt.Sprintf("wait for pod %s running again", fmt.Sprintf("%s-1", UID)))
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
				gomega.Expect(err).To(gomega.BeNil())
			}
		})
	})
})
