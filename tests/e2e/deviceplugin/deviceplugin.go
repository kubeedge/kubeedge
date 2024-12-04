/*
Copyright 2024 The KubeEdge Authors.

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

package deviceplugin

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var DevicePluginTestTimerGroup = utils.NewTestTimerGroup()

func newDevicePluginDeployment(name, imageURL string, replicas int32) *apps.Deployment {
	depl := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault, // Explicitly use default namespace
			Labels:    map[string]string{"app": name},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                 name,
					constants.E2ELabelKey: constants.E2ELabelValue,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                 name,
						constants.E2ELabelKey: constants.E2ELabelValue,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  name,
							Image: imageURL,
							SecurityContext: &v1.SecurityContext{
								Privileged: &[]bool{true}[0],
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "device-plugin",
									MountPath: "/var/lib/edged/device-plugins",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "device-plugin",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/edged/device-plugins",
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/edge": "",
					},
				},
			},
		},
	}
	return &depl
}

var _ = GroupDescribe("Device Plugin E2E Tests", func() {
	var UID string
	var testTimer *utils.TestTimer
	var testSpecReport ginkgo.SpecReport

	var clientSet clientset.Interface

	ginkgo.BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
	})

	ginkgo.Context("Test Device Plugin Registration and Basic Functionality", func() {
		ginkgo.BeforeEach(func() {
			// Get current test SpecReport
			testSpecReport = ginkgo.CurrentSpecReport()
			// Start test timer
			testTimer = DevicePluginTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
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
			labelSelector := labels.SelectorFromSet(map[string]string{
				"app":                 UID,
				constants.E2ELabelKey: constants.E2ELabelValue,
			})
			podList, err := utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(podList.Items).NotTo(gomega.BeEmpty(), "Pod list should not be empty")

			ginkgo.By(fmt.Sprintf("Deleting deployment %s", UID))
			err = utils.DeleteDeployment(clientSet, metav1.NamespaceDefault, deployment.Name)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("wait for pod of deploy %s to disappear", UID))
			err = utils.WaitForPodsToDisappear(clientSet, metav1.NamespaceDefault, labelSelector, constants.Interval, constants.Timeout)
			gomega.Expect(err).To(gomega.BeNil())

			utils.PrintTestcaseNameandStatus()
		})

		ginkgo.It("E2E_DEVICE_PLUGIN_1: Verify device plugin registration", func() {
			replica := int32(1)
			// Generate the random string and assign as a UID
			UID = "sample-device-plugin-" + utils.GetRandomString(5)

			ginkgo.By(fmt.Sprintf("Creating device plugin deployment %s", UID))
			deployment := newDevicePluginDeployment(UID, "nvidia/k8s-device-plugin:v0.13.0", replica)
			_, err := utils.CreateDeployment(clientSet, deployment)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Waiting for device plugin pod to be running")
			labelSelector := labels.SelectorFromSet(map[string]string{
				"app":                 UID,
				constants.E2ELabelKey: constants.E2ELabelValue,
			})
			podList, err := utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(podList.Items).NotTo(gomega.BeEmpty(), "Pod list should not be empty")
			
			utils.WaitForPodsRunning(clientSet, podList, 240*time.Second)

			ginkgo.By("Verifying device plugin registration")
			gomega.Eventually(func() bool {
				nodeList, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					framework.Logf("Error listing nodes: %v", err)
					return false
				}

				for _, node := range nodeList.Items {
					if _, ok := node.Status.Capacity["nvidia.com/gpu"]; ok {
						return true
					}
				}
				return false
			}, 5*time.Minute, 10*time.Second).Should(gomega.BeTrue(), "Device plugin should be registered on at least one node")

			framework.Logf("Device plugin successfully registered")
		})
	})
})
