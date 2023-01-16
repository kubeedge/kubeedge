/*
Copyright 2022 The KubeEdge Authors.

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

package controllermanager

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

const (
	pollInterval  = 1 * time.Second
	waitTimeOut   = 60 * time.Second
	locationLabel = "location"
	archAMD64     = "amd64"
)

var nodeTemplate *corev1.Node = &corev1.Node{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Node",
		APIVersion: "v1",
	},
	Status: corev1.NodeStatus{
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		},
	},
}

var _ = Describe("Test NodeGroup Controller", func() {
	var node1, node2 *corev1.Node
	var nodegroup1, nodegroup2 *appsv1alpha1.NodeGroup
	var randomize string

	BeforeEach(func() {
		randomize = uuid.New().String()
		node1 = nodeTemplate.DeepCopy()
		node1.Name = "node1-" + randomize
		node1.Labels = map[string]string{locationLabel: "location1-" + randomize}
		node2 = nodeTemplate.DeepCopy()
		node2.Name = "node2-" + randomize
		node2.Labels = map[string]string{locationLabel: "location2-" + randomize}
		nodegroup1 = &appsv1alpha1.NodeGroup{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeGroup",
				APIVersion: appsv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "location1-" + randomize,
			},
		}
		nodegroup2 = &appsv1alpha1.NodeGroup{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeGroup",
				APIVersion: appsv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "location2-" + randomize,
			},
		}
	})

	Context("Test Lifecycle Event", func() {
		When("create nodegroup", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
			})
			AfterEach(func() {
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
			})
			It("should be added with finalizer", func() {
				By("wait for the nodegroup to be added with finalizer")
				Eventually(func() bool {
					ng := &appsv1alpha1.NodeGroup{}
					ngKey := types.NamespacedName{Name: nodegroup1.Name}
					if err := k8sClient.Get(ctx, ngKey, ng); err != nil {
						return false
					}
					return reflect.DeepEqual(ng.Finalizers, []string{nodegroup.NodeGroupControllerFinalizer})
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
		})

		When("delete nodegroup", func() {
			BeforeEach(func() {
				By("creat one node in the cluster")
				Expect(k8sClient.Create(ctx, node1)).Should(Succeed())
				By("creat nodegroup in the cluster")
				nodegroup1.Spec.Nodes = []string{node1.Name}
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				By("wait for the node to be labeled")
				Eventually(func() bool {
					return beInMembership(node1.Name, nodegroup1.Name)
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			It("should remove all nodegroup labels on nodes and then remove its finalizer", func() {
				By("delete nodegroup")
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
				By("wait for the node to be unlabeled")
				Eventually(func() bool {
					return !beInMembership(node1.Name, nodegroup1.Name)
				}, waitTimeOut, pollInterval).Should(BeTrue())
				By("clear resources")
				Expect(k8sClient.Delete(ctx, node1)).Should(Succeed())
			})
		})
	})

	Context("Test Functionality", func() {
		BeforeEach(func() {
			By("create nodes")
			Expect(k8sClient.Create(ctx, node1)).Should(Succeed())
			Expect(k8sClient.Create(ctx, node2)).Should(Succeed())
		})
		AfterEach(func() {
			By("delete nodes")
			Expect(k8sClient.Delete(ctx, node1)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, node2)).Should(Succeed())
		})

		When("node changed", func() {
			BeforeEach(func() {
				By("create nodegroups")
				nodegroup1.Spec.MatchLabels = labelsDeepCopy(node1.Labels)
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				Eventually(func() bool {
					return beInMembership(node1.Name, nodegroup1.Name)
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					return havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			AfterEach(func() {
				By("delete nodegroups")
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
			})
			When("node label changed", func() {
				It("should add this node into nodegroup if new labels match MatchLabels", func() {
					old := node2.DeepCopy()
					node2.Labels = labelsDeepCopy(node1.Labels)
					Expect(k8sClient.Patch(ctx, node2, client.MergeFrom(old))).Should(Succeed())
					Eventually(func() bool {
						return beInMembership(node2.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						return havingStatusEntryAs(nodegroup1.Name, node2.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should remove this node from nodegroup and evict pods that can only run in this nodegroup if new labels does not match MatchLabels", func() {
					runningPod := &corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app" + randomize,
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							NodeName: node1.Name,
							NodeSelector: map[string]string{
								nodegroup.LabelBelongingTo: nodegroup1.Name,
							},
							Containers: []corev1.Container{
								{
									Name:  "foo",
									Image: "foo.io/foobar",
								},
							},
						},
					}
					Expect(k8sClient.Create(ctx, runningPod)).Should(Succeed())
					old := node1.DeepCopy()
					node1.Labels[locationLabel] = "unknown"
					Expect(k8sClient.Patch(ctx, node1, client.MergeFrom(old))).Should(Succeed())

					By("check if node has been removed from nodegroup")
					Eventually(func() bool {
						return beInMembership(node1.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeFalse())

					By("check if the status of this node has been removed from nodegroup")
					Eventually(func() bool {
						ng := &appsv1alpha1.NodeGroup{}
						key := types.NamespacedName{Name: nodegroup1.Name}
						if err := k8sClient.Get(ctx, key, ng); err != nil {
							return false
						}
						for _, s := range ng.Status.NodeStatuses {
							if s.NodeName == node1.Name {
								return false
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())

					By("check pods that should run in this nodegroup has been evicted from the node")
					Eventually(func() bool {
						pod := &corev1.Pod{}
						key := types.NamespacedName{Namespace: runningPod.Namespace, Name: runningPod.Name}
						err := k8sClient.Get(ctx, key, pod)
						return apierrors.IsNotFound(err) || !pod.DeletionTimestamp.IsZero()
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should be re-added with belongingTo label on node if it be removed", func() {
					old := node1.DeepCopy()
					delete(node1.Labels, nodegroup.LabelBelongingTo)
					Expect(k8sClient.Patch(ctx, node1, client.MergeFrom(old))).Should(Succeed())
					Eventually(func() bool {
						return beInMembership(node1.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
			})
			When("node ReadyCondition changed", func() {
				It("should update relative status entry for this node", func() {
					By("update node status")
					old := node1.DeepCopy()
					for i, condition := range node1.Status.Conditions {
						if condition.Type == corev1.NodeReady {
							node1.Status.Conditions[i].Status = corev1.ConditionFalse
							break
						}
					}
					Expect(k8sClient.Status().Patch(ctx, node1, client.MergeFrom(old))).Should(Succeed())

					By("check new node status in nodegroup")
					Eventually(func() bool {
						return havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeNotReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
			})
		})
		When("create nodegroup", func() {
			var node3, node4 *corev1.Node
			BeforeEach(func() {
				node3 = nodeTemplate.DeepCopy()
				node3.Name = "node3-" + randomize
				node3.Labels = labelsDeepCopy(node1.Labels)
				node4 = nodeTemplate.DeepCopy()
				node4.Name = "node4-" + randomize
				node4.Labels = labelsDeepCopy(node2.Labels)
				By("create additional nodes")
				Expect(k8sClient.Create(ctx, node3)).Should(Succeed())
				Expect(k8sClient.Create(ctx, node4)).Should(Succeed())
			})
			AfterEach(func() {
				By("delete additional nodes")
				Expect(k8sClient.Delete(ctx, node3)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, node4)).Should(Succeed())
			})

			It("should select nodes only with node names", func() {
				nodegroup1.Spec.Nodes = []string{node1.Name, node3.Name}
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node3.Name} {
						if !beInMembership(nodeName, nodegroup1.Name) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node3.Name} {
						if !havingStatusEntryAs(nodegroup1.Name, nodeName, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
			})
			It("should select nodes only with labels", func() {
				nodegroup1.Spec.MatchLabels = labelsDeepCopy(node1.Labels)
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node3.Name} {
						if !beInMembership(nodeName, nodegroup1.Name) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node3.Name} {
						if !havingStatusEntryAs(nodegroup1.Name, nodeName, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
			})
			It("should select nodes that match all labels", func() {
				old := node3.DeepCopy()
				node3.Labels["kubernetes.io/arch"] = archAMD64
				Expect(k8sClient.Patch(ctx, node3, client.MergeFrom(old))).Should(Succeed())

				nodegroup1.Spec.MatchLabels = labelsDeepCopy(node1.Labels)
				nodegroup1.Spec.MatchLabels["kubernetes.io/arch"] = archAMD64
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				Eventually(func() bool {
					return !beInMembership(node1.Name, nodegroup1.Name) && beInMembership(node3.Name, nodegroup1.Name)
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					return !havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) &&
						havingStatusEntryAs(nodegroup1.Name, node3.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, nodegroup1))
			})
			It("should select nodes with node names and labels simultaneously when they have no intersection", func() {
				nodegroup1.Spec.MatchLabels = labelsDeepCopy(node1.Labels)
				nodegroup1.Spec.Nodes = []string{node2.Name, node4.Name}
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node2.Name, node3.Name, node4.Name} {
						if !beInMembership(nodeName, nodegroup1.Name) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node2.Name, node3.Name, node4.Name} {
						if !havingStatusEntryAs(nodegroup1.Name, nodeName, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
			})
			It("should select nodes with node names and labels simultaneously when they have intersection", func() {
				nodegroup1.Spec.MatchLabels = labelsDeepCopy(node1.Labels)
				nodegroup1.Spec.Nodes = []string{node2.Name, node3.Name}
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node2.Name, node3.Name} {
						if !beInMembership(nodeName, nodegroup1.Name) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node2.Name, node3.Name} {
						if !havingStatusEntryAs(nodegroup1.Name, nodeName, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
			})
			It("should contain status entry for non-existing node that selected by node name", func() {
				nodegroup1.Spec.Nodes = []string{"non-existing-node"}
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				Eventually(func() bool {
					return havingStatusEntryAs(nodegroup1.Name, "non-existing-node", appsv1alpha1.Unknown, appsv1alpha1.FailedSelection)
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
			})
			It("should not select nodes that have already belonged to another nodegroup", func() {
				nodegroup1.Spec.MatchLabels = labelsDeepCopy(node1.Labels)
				Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node3.Name} {
						if !beInMembership(nodeName, nodegroup1.Name) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node3.Name} {
						if !havingStatusEntryAs(nodegroup1.Name, nodeName, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())

				By("create another nodegroup selecting with node name")
				nodegroup2.Spec.Nodes = []string{node1.Name}
				Expect(k8sClient.Create(ctx, nodegroup2)).Should(Succeed())
				Eventually(func() bool {
					return havingStatusEntryAs(nodegroup2.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.FailedSelection)
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					return beInMembership(node1.Name, nodegroup1.Name)
				}, waitTimeOut, pollInterval).Should(BeTrue())

				By("select nodes with label")
				old := nodegroup2.DeepCopy()
				nodegroup2.Spec.MatchLabels = labelsDeepCopy(node1.Labels)
				Expect(k8sClient.Patch(ctx, nodegroup2, client.MergeFrom(old))).Should(Succeed())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node3.Name} {
						if !havingStatusEntryAs(nodegroup2.Name, nodeName, appsv1alpha1.NodeReady, appsv1alpha1.FailedSelection) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					for _, nodeName := range []string{node1.Name, node3.Name} {
						if !beInMembership(nodeName, nodegroup1.Name) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, nodegroup2)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
			})
		})
		When("update nodegroup", func() {
			When("update node names of nodegroup", func() {
				BeforeEach(func() {
					By("create nodegroup")
					nodegroup1.Spec.Nodes = []string{node1.Name}
					Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
					Eventually(func() bool {
						return beInMembership(node1.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						return havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				AfterEach(func() {
					By("delete nodegroup")
					Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
				})
				It("should remove node from nodegroup if its node name has been removed", func() {
					old := nodegroup1.DeepCopy()
					nodegroup1.Spec.Nodes = []string{}
					Expect(k8sClient.Patch(ctx, nodegroup1, client.MergeFrom(old)))
					Eventually(func() bool {
						return !beInMembership(node1.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						return !havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should add node into nodegroup if its node name has been added", func() {
					old := nodegroup1.DeepCopy()
					nodegroup1.Spec.Nodes = append(nodegroup1.Spec.Nodes, node2.Name)
					Expect(k8sClient.Patch(ctx, nodegroup1, client.MergeFrom(old)))
					Eventually(func() bool {
						for _, name := range []string{node1.Name, node2.Name} {
							if !beInMembership(name, nodegroup1.Name) {
								return false
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						for _, name := range []string{node1.Name, node2.Name} {
							if !havingStatusEntryAs(nodegroup1.Name, name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) {
								return false
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should change the member nodes if nodes of nodegroup has been changed", func() {
					old := nodegroup1.DeepCopy()
					nodegroup1.Spec.Nodes = []string{node2.Name}
					Expect(k8sClient.Patch(ctx, nodegroup1, client.MergeFrom(old)))
					Eventually(func() bool {
						return !beInMembership(node1.Name, nodegroup1.Name) && beInMembership(node2.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						return !havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) &&
							havingStatusEntryAs(nodegroup1.Name, node2.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
			})
			When("update MatchLabels of nodegroup", func() {
				BeforeEach(func() {
					By("create nodegroup")
					nodegroup1.Spec.MatchLabels = labelsDeepCopy(node1.Labels)
					Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
					Eventually(func() bool {
						return beInMembership(node1.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						return havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				AfterEach(func() {
					By("delete nodegroup")
					Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
				})
				It("should remove nodes from nodegroup when removing label from MatchLabels", func() {
					old := nodegroup1.DeepCopy()
					nodegroup1.Spec.MatchLabels = map[string]string{}
					Expect(k8sClient.Patch(ctx, nodegroup1, client.MergeFrom(old))).Should(Succeed())
					Eventually(func() bool {
						return !beInMembership(node1.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						return !havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should reconcile nodes when adding label into MatchLabels", func() {
					node3 := nodeTemplate.DeepCopy()
					node3.Name = "node3" + randomize
					node3.Labels = labelsDeepCopy(node1.Labels)
					node3.Labels["kubernetes.io/arch"] = archAMD64
					Expect(k8sClient.Create(ctx, node3)).Should(Succeed())
					old := nodegroup1.DeepCopy()
					nodegroup1.Spec.MatchLabels["kubernetes.io/arch"] = archAMD64
					Expect(k8sClient.Patch(ctx, nodegroup1, client.MergeFrom(old))).Should(Succeed())
					Eventually(func() bool {
						return beInMembership(node3.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						return havingStatusEntryAs(nodegroup1.Name, node3.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Expect(k8sClient.Delete(ctx, node3)).Should(Succeed())
				})
				It("should change member nodes when changing MatchLabels", func() {
					node3 := nodeTemplate.DeepCopy()
					node3.Name = "node3" + randomize
					node3.Labels = map[string]string{
						"kubernetes.io/arch": archAMD64,
					}
					Expect(k8sClient.Create(ctx, node3)).Should(Succeed())
					old := nodegroup1.DeepCopy()
					nodegroup1.Spec.MatchLabels = labelsDeepCopy(node3.Labels)
					Expect(k8sClient.Patch(ctx, nodegroup1, client.MergeFrom(old))).Should(Succeed())
					Eventually(func() bool {
						return !beInMembership(node1.Name, nodegroup1.Name) && beInMembership(node3.Name, nodegroup1.Name)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Eventually(func() bool {
						return !havingStatusEntryAs(nodegroup1.Name, node1.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection) &&
							havingStatusEntryAs(nodegroup1.Name, node3.Name, appsv1alpha1.NodeReady, appsv1alpha1.SucceededSelection)
					}, waitTimeOut, pollInterval).Should(BeTrue())
					Expect(k8sClient.Delete(ctx, node3)).Should(Succeed())
				})
			})
		})
	})
})

var _ = Describe("Test EdgeApplication Controller", func() {
	var randomize string
	var deployTemplate *appsv1.Deployment
	var serviceTemplate *corev1.Service
	var configMapTemplate *corev1.ConfigMap
	var node1, node2, node3, node4 *corev1.Node
	var nodegroup1, nodegroup2 *appsv1alpha1.NodeGroup
	var edgeapp *appsv1alpha1.EdgeApplication
	BeforeEach(func() {
		randomize = uuid.New().String()
		node1 = nodeTemplate.DeepCopy()
		node1.Name = "node1-" + randomize
		node1.Labels = map[string]string{locationLabel: "location1-" + randomize}
		node2 = nodeTemplate.DeepCopy()
		node2.Name = "node2-" + randomize
		node2.Labels = map[string]string{locationLabel: "location1-" + randomize}
		node3 = nodeTemplate.DeepCopy()
		node3.Name = "node3-" + randomize
		node3.Labels = map[string]string{locationLabel: "location2-" + randomize}
		node4 = nodeTemplate.DeepCopy()
		node4.Name = "node4-" + randomize
		node4.Labels = map[string]string{locationLabel: "location2-" + randomize}
		nodegroup1 = &appsv1alpha1.NodeGroup{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeGroup",
				APIVersion: appsv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "location1-" + randomize,
			},
			Spec: appsv1alpha1.NodeGroupSpec{
				MatchLabels: map[string]string{
					locationLabel: "location1-" + randomize,
				},
			},
		}
		nodegroup2 = &appsv1alpha1.NodeGroup{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeGroup",
				APIVersion: appsv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "location2-" + randomize,
			},
			Spec: appsv1alpha1.NodeGroupSpec{
				MatchLabels: map[string]string{
					locationLabel: "location2-" + randomize,
				},
			},
		}
		serviceTemplate = &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apps-svc" + randomize,
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"label": "app",
				},
				Ports: []corev1.ServicePort{
					{
						TargetPort: intstr.IntOrString{
							IntVal: 8080,
						},
						Port: 8080,
					},
				},
			},
		}
		configMapTemplate = &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apps-cm" + randomize,
				Namespace: "default",
			},
			Data: map[string]string{
				"foo": "bar",
			},
		}
		deployTemplate = &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-deploy" + randomize,
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-pod",
						Namespace: "default",
						Labels: map[string]string{
							"label": "app",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container1",
								Image: "foo.fir.io/bar:latest",
							},
							{
								Name:  "container2",
								Image: "foo.sec.io/bar:v0.1.0",
							},
						},
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"label": "app",
					},
				},
				Replicas: pointer.Int32Ptr(1),
			},
		}
		edgeapp = &appsv1alpha1.EdgeApplication{
			TypeMeta: metav1.TypeMeta{
				Kind:       "EdgeApplication",
				APIVersion: appsv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "edge-app" + randomize,
				Namespace: "default",
			},
		}

		Expect(k8sClient.Create(ctx, node1)).Should(Succeed())
		Expect(k8sClient.Create(ctx, node2)).Should(Succeed())
		Expect(k8sClient.Create(ctx, node3)).Should(Succeed())
		Expect(k8sClient.Create(ctx, node4)).Should(Succeed())
		Expect(k8sClient.Create(ctx, nodegroup1)).Should(Succeed())
		Expect(k8sClient.Create(ctx, nodegroup2)).Should(Succeed())

		Eventually(func() bool {
			for _, nodeName := range []string{node1.Name, node2.Name} {
				if !beInMembership(nodeName, nodegroup1.Name) {
					return false
				}
			}
			return true
		}).Should(BeTrue())
		Eventually(func() bool {
			for _, nodeName := range []string{node3.Name, node4.Name} {
				if !beInMembership(nodeName, nodegroup2.Name) {
					return false
				}
			}
			return true
		}).Should(BeTrue())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, node1)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, node2)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, node3)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, node4)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, nodegroup1)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, nodegroup2)).Should(Succeed())
	})

	Context("Test Functionality", func() {
		BeforeEach(func() {
			edgeapp.Spec.WorkloadTemplate = appsv1alpha1.ResourceTemplate{
				Manifests: []appsv1alpha1.Manifest{
					{
						RawExtension: runtime.RawExtension{
							Object: deployTemplate,
						},
					},
					{
						RawExtension: runtime.RawExtension{
							Object: serviceTemplate,
						},
					},
					{
						RawExtension: runtime.RawExtension{
							Object: configMapTemplate,
						},
					},
				},
			}
			edgeapp.Spec.WorkloadScope = appsv1alpha1.WorkloadScope{
				TargetNodeGroups: []appsv1alpha1.TargetNodeGroup{
					{
						Name: nodegroup1.Name,
					},
					{
						Name: nodegroup2.Name,
					},
				},
			}
		})

		When("create edgeapplication", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, edgeapp)).Should(Succeed())
			})
			It("should create deployments for all nodegroups", func() {
				Eventually(func() bool {
					for _, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
						deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
						deploy := &appsv1.Deployment{}
						if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
							return false
						}
						if *deploy.Spec.Replicas != *deployTemplate.Spec.Replicas {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			It("should create service with topology annotation", func() {
				Eventually(func() bool {
					svc := &corev1.Service{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: serviceTemplate.Name}, svc); err != nil {
						return false
					}
					if svc.Annotations == nil {
						return false
					}
					v, ok := svc.Annotations[nodegroup.ServiceTopologyAnnotation]
					return ok && v == nodegroup.ServiceTopologyRangeNodegroup
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			It("should create the configmap as configmap template", func() {
				Eventually(func() bool {
					cm := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: configMapTemplate.Name}, cm); err != nil {
						return false
					}

					if !reflect.DeepEqual(cm.Data, configMapTemplate.Data) {
						return false
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
		})

		Context("Test Overrider", func() {
			When("add overrider", func() {
				BeforeEach(func() {
					Expect(k8sClient.Create(ctx, edgeapp)).Should(Succeed())
					Eventually(func() bool {
						for _, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
							deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
							deploy := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
								return false
							}
							if *deploy.Spec.Replicas != *deployTemplate.Spec.Replicas {
								return false
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should modify replicas of deployments for each nodegroup when adding replicas overrider", func() {
					newEdgeapp := edgeapp.DeepCopy()
					replicas := []int{10, 20}
					newEdgeapp.Spec.WorkloadScope.TargetNodeGroups[0].Overriders = appsv1alpha1.Overriders{
						Replicas: pointer.IntPtr(replicas[0]),
					}
					newEdgeapp.Spec.WorkloadScope.TargetNodeGroups[1].Overriders = appsv1alpha1.Overriders{
						Replicas: pointer.IntPtr(replicas[1]),
					}
					Expect(k8sClient.Patch(ctx, newEdgeapp, client.MergeFrom(edgeapp))).Should(Succeed())
					Eventually(func() bool {
						for i, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
							deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
							deploy := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
								return false
							}
							if int(*deploy.Spec.Replicas) != replicas[i] {
								return false
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should modify image of deployments for each nodegroup when adding image overrider", func() {
					imageOverriders := []appsv1alpha1.ImageOverrider{
						{
							Component: appsv1alpha1.Tag,
							Operator:  appsv1alpha1.OverriderOpReplace,
							Predicate: &appsv1alpha1.ImagePredicate{
								Path: "/spec/template/spec/containers/0/image",
							},
							Value: "new-value",
						},
						{
							Component: appsv1alpha1.Registry,
							Operator:  appsv1alpha1.OverriderOpRemove,
							Predicate: &appsv1alpha1.ImagePredicate{
								Path: "/spec/template/spec/containers/0/image",
							},
						},
						{
							Component: appsv1alpha1.Registry,
							Operator:  appsv1alpha1.OverriderOpRemove,
							Predicate: &appsv1alpha1.ImagePredicate{
								Path: "/spec/template/spec/containers/1/image",
							},
						},
					}

					newEdgeapp := edgeapp.DeepCopy()
					newEdgeapp.Spec.WorkloadScope.TargetNodeGroups[0].Overriders = appsv1alpha1.Overriders{
						ImageOverriders: imageOverriders,
					}
					Expect(k8sClient.Patch(ctx, newEdgeapp, client.MergeFrom(edgeapp))).Should(Succeed())
					Eventually(func() bool {
						expectImages := []string{
							"bar:new-value",
							"bar:v0.1.0",
						}
						deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, nodegroup1.Name)
						deploy := &appsv1.Deployment{}
						if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
							return false
						}
						for j, c := range deploy.Spec.Template.Spec.Containers {
							if c.Image != expectImages[j] {
								return false
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
			})
			When("update overrider", func() {
				var originReplicas []int
				var originImages [][]string
				BeforeEach(func() {
					originReplicas = []int{20, 30}
					originImages = [][]string{
						{"replaced-registry/bar:latest", "replaced-registry/bar"},
						{"foo.fir.io/bar-added:latest", "foo.sec.io/bar-added:v0.1.0"},
					}
					edgeapp.Spec.WorkloadScope.TargetNodeGroups[0].Overriders = appsv1alpha1.Overriders{
						Replicas: pointer.IntPtr(originReplicas[0]),
						ImageOverriders: []appsv1alpha1.ImageOverrider{
							{
								Component: appsv1alpha1.Registry,
								Operator:  appsv1alpha1.OverriderOpReplace,
								Value:     "replaced-registry",
							},
							{
								Component: appsv1alpha1.Tag,
								Operator:  appsv1alpha1.OverriderOpRemove,
								Predicate: &appsv1alpha1.ImagePredicate{
									Path: "/spec/template/spec/containers/1/image",
								},
							},
						},
					}
					edgeapp.Spec.WorkloadScope.TargetNodeGroups[1].Overriders = appsv1alpha1.Overriders{
						Replicas: pointer.IntPtr(originReplicas[1]),
						ImageOverriders: []appsv1alpha1.ImageOverrider{
							{
								Component: appsv1alpha1.Repository,
								Operator:  appsv1alpha1.OverriderOpAdd,
								Value:     "-added",
							},
						},
					}
					Expect(k8sClient.Create(ctx, edgeapp)).Should(Succeed())
					Eventually(func() bool {
						for i, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
							deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
							deploy := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
								return false
							}
							for j, c := range deploy.Spec.Template.Spec.Containers {
								if c.Image != originImages[i][j] {
									return false
								}
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should update replicas of deployments for each nodegroup when adding replicas overrider", func() {
					newEdgeApp := edgeapp.DeepCopy()
					newReplicas := []int{2, 3}
					newEdgeApp.Spec.WorkloadScope.TargetNodeGroups[0].Overriders.Replicas = &newReplicas[0]
					newEdgeApp.Spec.WorkloadScope.TargetNodeGroups[1].Overriders.Replicas = &newReplicas[1]
					Expect(k8sClient.Patch(ctx, newEdgeApp, client.MergeFrom(edgeapp))).Should(Succeed())
					Eventually(func() bool {
						for i, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
							deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
							deploy := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
								return false
							}
							if int(*deploy.Spec.Replicas) != newReplicas[i] {
								return false
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should update image of deployments for each nodegroup when adding image overrider", func() {
					newEdgeApp := edgeapp.DeepCopy()
					newImages := [][]string{
						{"bar:latest", "bar:latest"},
						{"foo.fir.io-added/bar:latest", "foo.sec.io-added/bar:v0.1.0"},
					}
					newEdgeApp.Spec.WorkloadScope.TargetNodeGroups[0].Overriders.ImageOverriders = []appsv1alpha1.ImageOverrider{
						{
							Component: appsv1alpha1.Registry,
							Operator:  appsv1alpha1.OverriderOpRemove,
						},
						{
							Component: appsv1alpha1.Tag,
							Operator:  appsv1alpha1.OverriderOpReplace,
							Predicate: &appsv1alpha1.ImagePredicate{
								Path: "/spec/template/spec/containers/1/image",
							},
							Value: "latest",
						},
					}
					newEdgeApp.Spec.WorkloadScope.TargetNodeGroups[1].Overriders.ImageOverriders = []appsv1alpha1.ImageOverrider{
						{
							Component: appsv1alpha1.Registry,
							Operator:  appsv1alpha1.OverriderOpAdd,
							Value:     "-added",
						},
					}
					Expect(k8sClient.Patch(ctx, newEdgeApp, client.MergeFrom(edgeapp))).Should(Succeed())
					Eventually(func() bool {
						for i, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
							deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
							deploy := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
								return false
							}
							for j, c := range deploy.Spec.Template.Spec.Containers {
								if c.Image != newImages[i][j] {
									return false
								}
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
			})
			When("delete overrider", func() {
				var originReplicas []int
				var originImages [][]string
				BeforeEach(func() {
					originReplicas = []int{20, 30}
					originImages = [][]string{
						{"replaced-registry/bar:latest", "replaced-registry/bar"},
						{"foo.fir.io/bar-added:latest", "foo.sec.io/bar-added:v0.1.0"},
					}
					edgeapp.Spec.WorkloadScope.TargetNodeGroups[0].Overriders = appsv1alpha1.Overriders{
						Replicas: pointer.IntPtr(originReplicas[0]),
						ImageOverriders: []appsv1alpha1.ImageOverrider{
							{
								Component: appsv1alpha1.Registry,
								Operator:  appsv1alpha1.OverriderOpReplace,
								Value:     "replaced-registry",
							},
							{
								Component: appsv1alpha1.Tag,
								Operator:  appsv1alpha1.OverriderOpRemove,
								Predicate: &appsv1alpha1.ImagePredicate{
									Path: "/spec/template/spec/containers/1/image",
								},
							},
						},
					}
					edgeapp.Spec.WorkloadScope.TargetNodeGroups[1].Overriders = appsv1alpha1.Overriders{
						Replicas: pointer.IntPtr(originReplicas[1]),
						ImageOverriders: []appsv1alpha1.ImageOverrider{
							{
								Component: appsv1alpha1.Repository,
								Operator:  appsv1alpha1.OverriderOpAdd,
								Value:     "-added",
							},
						},
					}
					Expect(k8sClient.Create(ctx, edgeapp)).Should(Succeed())
					Eventually(func() bool {
						for i, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
							deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
							deploy := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
								return false
							}
							if int(*deploy.Spec.Replicas) != originReplicas[i] {
								return false
							}
							for j, c := range deploy.Spec.Template.Spec.Containers {
								if c.Image != originImages[i][j] {
									return false
								}
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should reset the replicas of deployments when removing replicas overrider", func() {
					newEdgeApp := edgeapp.DeepCopy()
					newEdgeApp.Spec.WorkloadScope.TargetNodeGroups[0].Overriders.Replicas = nil
					newEdgeApp.Spec.WorkloadScope.TargetNodeGroups[1].Overriders.Replicas = nil
					Expect(k8sClient.Patch(ctx, newEdgeApp, client.MergeFrom(edgeapp))).Should(Succeed())
					Eventually(func() bool {
						for _, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
							deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
							deploy := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
								return false
							}
							if *deployTemplate.Spec.Replicas != *deploy.Spec.Replicas {
								return false
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
				It("should reset the image of deployments when removing image overrider", func() {
					newEdgeApp := edgeapp.DeepCopy()
					newImages := [][]string{
						{"foo.fir.io/replaced-repo:latest", "foo.sec.io/replaced-repo:v0.1.0"},
						{"foo.fir.io/bar:latest", "foo.sec.io/bar:v0.1.0"},
					}
					newEdgeApp.Spec.WorkloadScope.TargetNodeGroups[0].Overriders.ImageOverriders = []appsv1alpha1.ImageOverrider{
						{
							Component: appsv1alpha1.Repository,
							Operator:  appsv1alpha1.OverriderOpReplace,
							Value:     "replaced-repo",
						},
					}
					newEdgeApp.Spec.WorkloadScope.TargetNodeGroups[1].Overriders.ImageOverriders = []appsv1alpha1.ImageOverrider{}
					Expect(k8sClient.Patch(ctx, newEdgeApp, client.MergeFrom(edgeapp))).Should(Succeed())
					Eventually(func() bool {
						for i, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
							deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
							deploy := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
								return false
							}
							for j, c := range deploy.Spec.Template.Spec.Containers {
								if c.Image != newImages[i][j] {
									return false
								}
							}
						}
						return true
					}, waitTimeOut, pollInterval).Should(BeTrue())
				})
			})
		})

		When("update edgeapplication", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, edgeapp)).Should(Succeed())
				Eventually(func() bool {
					for _, ng := range []string{nodegroup1.Name, nodegroup2.Name} {
						deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ng)
						deploy := &appsv1.Deployment{}
						if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
							return false
						}
					}

					cm := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: configMapTemplate.Name}, cm); err != nil {
						return false
					}

					svc := &corev1.Service{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: serviceTemplate.Name}, svc); err != nil {
						return false
					}

					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			It("should update sub-resources when templates have been modified", func() {
				newEdgeApp := edgeapp.DeepCopy()
				newDeploy := deployTemplate.DeepCopy()
				newDeploy.Labels = map[string]string{"new-added": "true"}
				newService := serviceTemplate.DeepCopy()
				newService.Spec.Selector["new-added"] = "true"
				newConfigMap := configMapTemplate.DeepCopy()
				newConfigMap.Name = "updated-cm" + randomize
				newEdgeApp.Spec.WorkloadTemplate.Manifests = []appsv1alpha1.Manifest{
					{
						RawExtension: runtime.RawExtension{
							Object: newDeploy,
						},
					},
					{
						RawExtension: runtime.RawExtension{
							Object: newService,
						},
					},
					{
						RawExtension: runtime.RawExtension{
							Object: newConfigMap,
						},
					},
				}
				Expect(k8sClient.Patch(ctx, newEdgeApp, client.MergeFrom(edgeapp))).Should(Succeed())
				Eventually(func() bool {
					for _, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
						deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
						deploy := &appsv1.Deployment{}
						if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
							return false
						}
						if !reflect.DeepEqual(deploy.Labels, newDeploy.Labels) {
							return false
						}
					}

					svc := &corev1.Service{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: serviceTemplate.Name}, svc); err != nil {
						return false
					}
					if !reflect.DeepEqual(svc.Spec.Selector, newService.Spec.Selector) {
						return false
					}

					cm := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: newConfigMap.Name}, cm); err != nil {
						return false
					}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: configMapTemplate.Name}, cm); !apierrors.IsNotFound(err) {
						return false
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			It("should add/remove deployments for nodegroups when update names of targetNodeGroup", func() {
				newEdgeApp := edgeapp.DeepCopy()
				newNodeGroupName := []string{nodegroup1.Name, "newng1-" + randomize}
				newEdgeApp.Spec.WorkloadScope.TargetNodeGroups = []appsv1alpha1.TargetNodeGroup{
					{
						Name: newNodeGroupName[0],
					},
					{
						Name: newNodeGroupName[1],
					},
				}
				Expect(k8sClient.Patch(ctx, newEdgeApp, client.MergeFrom(edgeapp))).Should(Succeed())
				Eventually(func() bool {
					for _, ng := range newNodeGroupName {
						deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ng)
						deploy := &appsv1.Deployment{}
						if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, nodegroup2.Name)
					deploy := &appsv1.Deployment{}
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy)
					return apierrors.IsNotFound(err) || !deploy.DeletionTimestamp.IsZero()
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
		})
	})

	Context("Test Lifecycle Event", func() {
		BeforeEach(func() {
			location1Replicas := 2
			location2Replicas := 3
			edgeapp.Spec.WorkloadTemplate = appsv1alpha1.ResourceTemplate{
				Manifests: []appsv1alpha1.Manifest{
					{
						RawExtension: runtime.RawExtension{
							Object: deployTemplate,
						},
					},
					{
						RawExtension: runtime.RawExtension{
							Object: serviceTemplate,
						},
					},
					{
						RawExtension: runtime.RawExtension{
							Object: configMapTemplate,
						},
					},
				},
			}
			edgeapp.Spec.WorkloadScope = appsv1alpha1.WorkloadScope{
				TargetNodeGroups: []appsv1alpha1.TargetNodeGroup{
					{
						Name: nodegroup1.Name,
						Overriders: appsv1alpha1.Overriders{
							Replicas: &location1Replicas,
						},
					},
					{
						Name: nodegroup2.Name,
						Overriders: appsv1alpha1.Overriders{
							Replicas: &location2Replicas,
						},
					},
				},
			}
		})
		When("create edgeapplication", func() {
			It("all sub-resources are added with OwnerReference", func() {
				Expect(k8sClient.Create(ctx, edgeapp)).Should(Succeed())
				Eventually(func() bool {
					for _, ngName := range []string{nodegroup1.Name, nodegroup2.Name} {
						deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ngName)
						deploy := &appsv1.Deployment{}
						if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
							return false
						}
						if !isOwnedByEdgeApp(deploy.OwnerReferences, edgeapp) {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					svc := &corev1.Service{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: serviceTemplate.Name}, svc); err != nil {
						return false
					}
					if !isOwnedByEdgeApp(svc.OwnerReferences, edgeapp) {
						return false
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					cm := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: configMapTemplate.Name}, cm); err != nil {
						return false
					}
					if !isOwnedByEdgeApp(cm.OwnerReferences, edgeapp) {
						return false
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
		})
		When("delete edgeapplication", func() {
			It("should delete all sub-resources", func() {
				// TODO:
				// Currently, testEnv seems cannot simulate the cascading deletion of owner reference.
				// So this test cannot run correctly.
			})
		})
		When("delete sub-resources", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, edgeapp)).Should(Succeed())
				Eventually(func() bool {
					for _, ng := range []string{nodegroup1.Name, nodegroup2.Name} {
						deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ng)
						deploy := &appsv1.Deployment{}
						if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					svc := &corev1.Service{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: serviceTemplate.Name}, svc); err != nil {
						return false
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
				Eventually(func() bool {
					cm := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: configMapTemplate.Name}, cm); err != nil {
						return false
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			It("should recreated deployment automatically", func() {
				deployNG1 := deployTemplate.DeepCopy()
				deployNG2 := deployTemplate.DeepCopy()
				deployNG1.Name = fmt.Sprintf("%s-%s", deployTemplate.Name, nodegroup1.Name)
				deployNG2.Name = fmt.Sprintf("%s-%s", deployTemplate.Name, nodegroup2.Name)
				Expect(k8sClient.Delete(ctx, deployNG1)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, deployNG2)).Should(Succeed())
				Eventually(func() bool {
					for _, ng := range []string{nodegroup1.Name, nodegroup2.Name} {
						deployName := fmt.Sprintf("%s-%s", deployTemplate.Name, ng)
						deploy := &appsv1.Deployment{}
						if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: deployName}, deploy); err != nil {
							return false
						}
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			It("should recreated configmap automatically", func() {
				configmap := configMapTemplate.DeepCopy()
				Expect(k8sClient.Delete(ctx, configmap)).Should(Succeed())
				Eventually(func() bool {
					cm := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: configmap.Name}, cm); err != nil {
						return false
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
			It("should recreated service automatically", func() {
				service := serviceTemplate.DeepCopy()
				Expect(k8sClient.Delete(ctx, service)).Should(Succeed())
				Eventually(func() bool {
					svc := &corev1.Service{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: service.Name}, svc); err != nil {
						return false
					}
					return true
				}, waitTimeOut, pollInterval).Should(BeTrue())
			})
		})
	})
})

func labelsDeepCopy(src map[string]string) map[string]string {
	copy := map[string]string{}
	for k, v := range src {
		copy[k] = v
	}
	return copy
}

func beInMembership(nodeName string, nodeGroupName string) bool {
	node := &corev1.Node{}
	key := types.NamespacedName{Name: nodeName}
	if err := k8sClient.Get(ctx, key, node); err != nil {
		return false
	}
	v, ok := node.Labels[nodegroup.LabelBelongingTo]
	return ok && v == nodeGroupName
}

func havingStatusEntryAs(nodeGroupName string, nodeName string, ready appsv1alpha1.ReadyStatus, selection appsv1alpha1.SelectionStatus) bool {
	ng := &appsv1alpha1.NodeGroup{}
	key := types.NamespacedName{Name: nodeGroupName}
	if err := k8sClient.Get(ctx, key, ng); err != nil {
		return false
	}
	for _, s := range ng.Status.NodeStatuses {
		if s.NodeName == nodeName {
			return s.ReadyStatus == ready && s.SelectionStatus == selection
		}
	}
	return false
}

func isOwnedByEdgeApp(ownerReferences []metav1.OwnerReference, edgeApp *appsv1alpha1.EdgeApplication) bool {
	for _, o := range ownerReferences {
		if o.APIVersion == appsv1alpha1.GroupVersion.String() &&
			o.Kind == "EdgeApplication" &&
			*o.BlockOwnerDeletion == true &&
			*o.Controller == true &&
			o.Name == edgeApp.Name &&
			o.UID == edgeApp.UID {
			return true
		}
	}
	return false
}
