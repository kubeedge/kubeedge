package deviceplugin

import (
	"context"
	"fmt"
	"time"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	// apps "k8s.io/api/apps/v1"
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

	ginkgo.Context("Device Plugin basic test", func() {
		ginkgo.BeforeEach(func() {
			testSpecReport = ginkgo.CurrentSpecReport()
			testTimer = DevicePluginTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})

		ginkgo.AfterEach(func() {
			testTimer.End()
			testTimer.PrintResult()

			// if UID != "" {
			// 	ginkgo.By(fmt.Sprintf("Deleting deployment %s", UID))
			// 	err := utils.DeleteDeployment(clientSet, metav1.NamespaceDefault, UID)
			// 	gomega.Expect(err).To(gomega.BeNil())

			// 	labelSelector := labels.SelectorFromSet(map[string]string{
			// 		"app":                 UID,
			// 		constants.E2ELabelKey: constants.E2ELabelValue,
			// 	})

			// 	ginkgo.By(fmt.Sprintf("wait for pod of deploy %s to disappear", UID))
			// 	err = utils.WaitForPodsToDisappear(clientSet, metav1.NamespaceDefault, labelSelector, constants.Interval, constants.Timeout)
			// 	gomega.Expect(err).To(gomega.BeNil())
			// }

			ginkgo.By(fmt.Sprintf("Cleaning up device plugin pod %s", UID))
			err := utils.DeletePod(clientSet, v1.NamespaceDefault, UID)
			gomega.Expect(err).To(gomega.BeNil())

			utils.PrintTestcaseNameandStatus()
		})

		ginkgo.It("E2E_DEVICE_PLUGIN_1: Deploy device plugin and verify it registers devices", func() {
			// Step 1: Create the device plugin pod
			UID = "sample-device-plugin-" + utils.GetRandomString(5)
			devicePluginName := UID
			ginkgo.By(fmt.Sprintf("Creating device plugin pod %s", devicePluginName))
			devicePlugin := NewDevicePluginPod(devicePluginName, "nvidia/k8s-device-plugin:v0.5.0")
			_, err := utils.CreatePod(clientSet, devicePlugin)
			gomega.Expect(err).To(gomega.BeNil())

			// Step 2: Wait for the device plugin pod to be running
			ginkgo.By("Waiting for device plugin pod to be running")
			labelSelector := labels.SelectorFromSet(map[string]string{"app": devicePluginName})
			podList, err := utils.GetPods(clientSet, v1.NamespaceDefault, labelSelector, nil)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(podList.Items)).To(gomega.Equal(1))

			utils.WaitForPodsRunning(clientSet, podList, 240*time.Second)

			// Step 3: Give some time for the device plugin to register
			ginkgo.By("Waiting for device plugin registration")
			time.Sleep(60 * time.Second)

			// Get pod logs
			logs, _ := clientSet.CoreV1().Pods(v1.NamespaceDefault).GetLogs(devicePluginName, &v1.PodLogOptions{}).Do(context.TODO()).Raw()
			framework.Logf("Device Plugin Pod Logs: %s", string(logs))		

			// Step 4: Verify device registration on the node
			ginkgo.By("Verifying device registration on the edge node")
			nodes, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
				LabelSelector: "node-role.kubernetes.io/edge",
			})
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(nodes.Items)).To(gomega.BeNumerically(">", 0), "No edge nodes found")

			ginkgo.By("Now Node status capacity")

			// Check if the device is registered in node capacity
			node := nodes.Items[0]
			framework.Logf("Node Name: %s", node.Name)
    		framework.Logf("Node Capacity: %v", node.Status.Capacity)
    		framework.Logf("Node Allocatable: %v", node.Status.Allocatable)

			// Check all available extended resources
			framework.Logf("Available Extended Resources:")
			for resourceName := range node.Status.Capacity {
				if strings.Contains(resourceName.String(), "/") {
					framework.Logf("Found Extended Resource: %s", resourceName)
				}
			}

			_, hasDevice := node.Status.Capacity["nvidia.com/gpu"] // Replace with your device type
			gomega.Expect(hasDevice).To(gomega.BeTrue(), "Device not registered on node")
		})

		// ginkgo.It("E2E_DEVICE_PLUGIN_1: Verify device plugin registration", func() {
		// 	replica := int32(1)
		// 	UID = "sample-device-plugin-" + utils.GetRandomString(5)

		// 	ginkgo.By(fmt.Sprintf("Creating device plugin deployment %s", UID))
		// 	deployment := newDevicePluginDeployment(UID, "nvidia/k8s-device-plugin:v0.13.0", replica)
			
		// 	framework.Logf("Deployment Namespace: %s", deployment.Namespace)
		// 	framework.Logf("Deployment Name: %s", deployment.Name)
		// 	framework.Logf("Deployment Labels: %v", deployment.Labels)
		// 	framework.Logf("Pod Template Labels: %v", deployment.Spec.Template.Labels)

		// 	createdDeployment, err := utils.CreateDeployment(clientSet, deployment)
		// 	gomega.Expect(err).To(gomega.BeNil())
		// 	gomega.Expect(createdDeployment).NotTo(gomega.BeNil())

		// 	ginkgo.By("Waiting for device plugin pod to be running")
		// 	labelSelector := labels.SelectorFromSet(map[string]string{
		// 		"app":                 UID,
		// 		constants.E2ELabelKey: constants.E2ELabelValue,
		// 	})

		// 	ginkgo.By("Retrieving pods with label selector")
		// 	var podList *v1.PodList
		// 	gomega.Eventually(func() bool {
		// 		var err error
		// 		podList, err = clientSet.CoreV1().Pods(metav1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{
		// 			LabelSelector: labelSelector.String(),
		// 		})
		// 		if err != nil {
		// 			framework.Logf("Error listing pods: %v", err)
		// 			return false
		// 		}
		// 		return len(podList.Items) > 0
		// 	}, 5*time.Minute, 10*time.Second).Should(gomega.BeTrue(), "Pod list should not be empty")

		// 	framework.Logf("Found %d pods", len(podList.Items))
		// 	for _, pod := range podList.Items {
		// 		framework.Logf("Pod Name: %s, Status: %s, Labels: %v", 
		// 			pod.Name, pod.Status.Phase, pod.Labels)
		// 	}
			
		// 	utils.WaitForPodsRunning(clientSet, podList, 5*time.Minute)

		// 	ginkgo.By("Verifying device plugin registration")
		// 	gomega.Eventually(func() bool {
		// 		nodeList, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		// 		if err != nil {
		// 			framework.Logf("Error listing nodes: %v", err)
		// 			return false
		// 		}

		// 		for _, node := range nodeList.Items {
		// 			if _, ok := node.Status.Capacity["nvidia.com/gpu"]; ok {
		// 				return true
		// 			}
		// 		}
		// 		return false
		// 	}, 10*time.Minute, 10*time.Second).Should(gomega.BeTrue(), "Device plugin should be registered on at least one node")

		// 	framework.Logf("Device plugin successfully registered")
		// })
	})
})

func NewDevicePluginPod(podName, imgURL string) *v1.Pod {
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"app":                 podName,
				constants.E2ELabelKey: constants.E2ELabelValue,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  podName,
					Image: imgURL,
					SecurityContext: &v1.SecurityContext{
						Privileged: &[]bool{true}[0],
					},
					VolumeMounts: []v1.VolumeMount{
                        {
                            Name:      "device-plugin",
                            MountPath: "/var/lib/kubelet/device-plugins",
                        },
                        {
                            Name:      "dev",
                            MountPath: "/dev",
                        },
                    },
				},
			},
			Volumes: []v1.Volume{
				{
                    Name: "device-plugin",
                    VolumeSource: v1.VolumeSource{
                        HostPath: &v1.HostPathVolumeSource{
                            Path: "/var/lib/kubelet/device-plugins",
                        },
                    },
                },
                {
                    Name: "dev",
                    VolumeSource: v1.VolumeSource{
                        HostPath: &v1.HostPathVolumeSource{
                            Path: "/dev",
                        },
                    },
                },
			},
			NodeSelector: map[string]string{
				"node-role.kubernetes.io/edge": "",
			},
		},
	}
	return &pod
}

// func newDevicePluginDeployment(name, imageURL string, replicas int32) *apps.Deployment {
// 	privileged := true
// 	depl := apps.Deployment{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: metav1.NamespaceDefault,
// 			Labels: map[string]string{
// 				"app":                 name,
// 				constants.E2ELabelKey: constants.E2ELabelValue,
// 			},
// 		},
// 		Spec: apps.DeploymentSpec{
// 			Replicas: &replicas,
// 			Selector: &metav1.LabelSelector{
// 				MatchLabels: map[string]string{
// 					"app":                 name,
// 					constants.E2ELabelKey: constants.E2ELabelValue,
// 				},
// 			},
// 			Template: v1.PodTemplateSpec{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Labels: map[string]string{
// 						"app":                 name,
// 						constants.E2ELabelKey: constants.E2ELabelValue,
// 					},
// 				},
// 				Spec: v1.PodSpec{
// 					Containers: []v1.Container{
// 						{
// 							Name:  name,
// 							Image: imageURL,
// 							SecurityContext: &v1.SecurityContext{
// 								Privileged: &privileged,
// 							},
// 							VolumeMounts: []v1.VolumeMount{
// 								{
// 									Name:      "device-plugin",
// 									MountPath: "/var/lib/edged/device-plugins",
// 								},
// 							},
// 						},
// 					},
// 					Volumes: []v1.Volume{
// 						{
// 							Name: "device-plugin",
// 							VolumeSource: v1.VolumeSource{
// 								HostPath: &v1.HostPathVolumeSource{
// 									Path: "/var/lib/edged/device-plugins",
// 								},
// 							},
// 						},
// 					},
// 					NodeSelector: map[string]string{
// 						"node-role.kubernetes.io/edge": "",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	return &depl
// }