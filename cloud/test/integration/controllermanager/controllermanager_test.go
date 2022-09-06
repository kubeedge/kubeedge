package controllermanager

import (
	"reflect"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	var service *corev1.Service
	var configMap *corev1.ConfigMap
	var node1, node2, node3, node4 *corev1.Node
	var nodegroup1, nodegroup2 *appsv1alpha1.NodeGroup
	BeforeEach(func() {
		randomize := uuid.New().String()
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

	})

	Context("Test Functionality", func() {

	})

	Context("Test Lifecycle Event", func() {
		When("create edgeappliction", func() {
			It("should be added with finalizer", func() {

			})
			It("all sub-resources are added with OwnerReference", func() {

			})
		})
		When("delete edgeapplication", func() {
			It("should delete all sub-resources", func() {

			})
		})
		When("modify sub-resources", func() {
			It("should not update modified sub-resources", func() {

			})
		})
		When("delete sub-resources", func() {
			It("should recreated deployment automatically", func() {

			})
			It("should recreated configmap automatically", func() {

			})
			It("should recreated service automatically", func() {

			})
		})
		When("node membership changed", func() {
			It("node selector of deployments should be changed", func() {

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
