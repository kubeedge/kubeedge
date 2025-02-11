
package deviceplugin

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var DevicePluginTestTimerGroup = utils.NewTestTimerGroup()

var _ = GroupDescribe("Device Plugin E2E Tests", func() {
	var UID string
	var testTimer *utils.TestTimer
	var testSpecReport ginkgo.SpecReport

	var clientSet clientset.Interface

	ginkgo.BeforeEach(func() {
		clientSet = utils.NewKubeClient(framework.TestContext.KubeConfig)
	})

	ginkgo.Context("Device Plugin Registration Test", func() {
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

			ginkgo.By("List of pods")
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			_, err := utils.GetPods(clientSet, metav1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("Cleaning up device plugin pod %s", UID))
			err = utils.DeletePod(clientSet, v1.NamespaceDefault, UID)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By(fmt.Sprintf("wait for device plugin pod %s to disappear", UID))
			err = utils.WaitForPodsToDisappear(clientSet, metav1.NamespaceDefault, labelSelector, constants.Interval, constants.Timeout)
			gomega.Expect(err).To(gomega.BeNil())

			utils.PrintTestcaseNameandStatus()
		})

		ginkgo.It("E2E_DEVICE_PLUGIN_1: Deploy device plugin pod and verify it registers devices", func() {
			// Create the device plugin pod
			UID = "sample-device-plugin-" + utils.GetRandomString(5)

			ginkgo.By(fmt.Sprintf("Creating device plugin pod %s", UID))
			devicePlugin := utils.NewDevicePluginPod(UID, "opsdockerimage/e2e-test-images-sample-device-plugin:1.7")
			_, err := utils.CreatePod(clientSet, devicePlugin)
			gomega.Expect(err).To(gomega.BeNil())

			// Wait for the device plugin pod to be running
			ginkgo.By("Waiting for device plugin pod to be running")
			labelSelector := labels.SelectorFromSet(map[string]string{"app": UID})
			podList, err := utils.GetPods(clientSet, v1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(podList.Items)).To(gomega.Equal(1))
			utils.WaitForPodsRunning(clientSet, podList, 240*time.Second)

			// Give some time for the device plugin to register
			ginkgo.By("Waiting for device plugin registration")
			time.Sleep(60 * time.Second)

			// Verify device registration on the node
			ginkgo.By("Verifying device registration on the edge node")
			nodes, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
				LabelSelector: "node-role.kubernetes.io/edge",
			})
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(nodes.Items)).To(gomega.BeNumerically(">", 0), "No edge nodes found")

			// Check if the device is registered in node capacity
			node := nodes.Items[0]

			expectedDevice := "example.com/resource"
			capacity, hasDevice := node.Status.Capacity[v1.ResourceName(expectedDevice)]
			allocatable, allocatableHasResource := node.Status.Allocatable[v1.ResourceName(expectedDevice)]

			// Log the results
			framework.Logf("Capacity for Device %s: %v", expectedDevice, capacity)
			framework.Logf("Allocatable for Device %s: %v", expectedDevice, allocatable)

			// Verify that the device is found
			gomega.Expect(hasDevice).To(gomega.BeTrue(), "Device not registered on node")
			gomega.Expect(allocatableHasResource).To(gomega.BeTrue(), fmt.Sprintf("Device %s not found in node allocatable", expectedDevice))
		})
	})
})