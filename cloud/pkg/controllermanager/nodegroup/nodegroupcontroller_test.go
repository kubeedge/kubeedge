/*
Copyright 2025 The KubeEdge Authors.

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

package nodegroup

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
)

func init() {
	_ = appsv1alpha1.AddToScheme(scheme.Scheme)
}

type MockClient struct {
	client.Client
	shouldErrorOn    map[string]bool
	capturedStatuses map[string][]appsv1alpha1.NodeStatus
}

func NewMockClient(objects ...runtime.Object) *MockClient {
	builder := fake.NewClientBuilder().WithIndex(&corev1.Pod{}, "spec.nodeName", func(obj client.Object) []string {
		pod := obj.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	})

	return &MockClient{
		Client:           builder.WithRuntimeObjects(objects...).Build(),
		shouldErrorOn:    make(map[string]bool),
		capturedStatuses: make(map[string][]appsv1alpha1.NodeStatus),
	}
}

func (c *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	if c.shouldErrorOn["get"] {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "test"}, key.Name)
	}
	return c.Client.Get(ctx, key, obj)
}

func (c *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if c.shouldErrorOn["list"] {
		return errors.New("mocked list error")
	}
	return c.Client.List(ctx, list, opts...)
}

func (c *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.shouldErrorOn["update"] {
		return errors.New("mocked update error")
	}
	return c.Client.Update(ctx, obj, opts...)
}

func (c *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if c.shouldErrorOn["patch"] {
		return errors.New("mocked patch error")
	}
	return c.Client.Patch(ctx, obj, patch, opts...)
}

func (c *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if c.shouldErrorOn["delete"] {
		return errors.New("mocked delete error")
	}
	return c.Client.Delete(ctx, obj, opts...)
}

func (c *MockClient) Status() client.StatusWriter {
	return &mockStatusWriter{
		StatusWriter: c.Client.Status(),
		mockClient:   c,
	}
}

type mockStatusWriter struct {
	client.StatusWriter
	mockClient *MockClient
}

func (sw *mockStatusWriter) Update(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
	if ng, ok := obj.(*appsv1alpha1.NodeGroup); ok {
		sw.mockClient.capturedStatuses[ng.Name] = ng.Status.NodeStatuses
	}
	return nil
}

func createNG(name string, matchLabels map[string]string, nodeNames []string) *appsv1alpha1.NodeGroup {
	return &appsv1alpha1.NodeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Finalizers: []string{NodeGroupControllerFinalizer},
		},
		Spec: appsv1alpha1.NodeGroupSpec{
			MatchLabels: matchLabels,
			Nodes:       nodeNames,
		},
	}
}

func createNode(name string, labels map[string]string, isReady bool) *corev1.Node {
	status := corev1.ConditionFalse
	if isReady {
		status = corev1.ConditionTrue
	}

	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: status},
			},
		},
	}
}

func createPod(name, namespace, nodeName string, nodeSelector map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeName:     nodeName,
			NodeSelector: nodeSelector,
		},
	}
}
func TestHelperFunctions(t *testing.T) {
	t.Run("NodesDiff", func(t *testing.T) {
		oldNodes := []corev1.Node{
			*createNode("node1", nil, true),
			*createNode("node2", nil, true),
		}

		newNodes := []corev1.Node{
			*createNode("node2", nil, true),
			*createNode("node3", nil, true),
		}

		deleted, added := nodesDiff(oldNodes, newNodes)

		assert.Len(t, deleted, 1)
		assert.Equal(t, "node1", deleted[0].Name)

		assert.Len(t, added, 1)
		assert.Equal(t, "node3", added[0].Name)
	})

	t.Run("NodesUnion", func(t *testing.T) {
		testCases := []struct {
			name  string
			list1 []corev1.Node
			list2 []corev1.Node
			want  int
		}{
			{"nil-nil", nil, nil, 0},
			{"nil-one", nil, []corev1.Node{*createNode("node1", nil, true)}, 1},
			{"two-three",
				[]corev1.Node{*createNode("node1", nil, true), *createNode("node2", nil, true)},
				[]corev1.Node{*createNode("node2", nil, true), *createNode("node3", nil, true), *createNode("node4", nil, true)},
				4},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := nodesUnion(tc.list1, tc.list2)
				assert.Len(t, result, tc.want)
			})
		}
	})

	t.Run("GetNodeReadyConditionFromNode", func(t *testing.T) {
		readyNode := createNode("node1", nil, true)
		status, found := getNodeReadyConditionFromNode(readyNode)
		assert.Equal(t, corev1.ConditionTrue, status)
		assert.True(t, found)

		notReadyNode := createNode("node2", nil, false)
		status, found = getNodeReadyConditionFromNode(notReadyNode)
		assert.Equal(t, corev1.ConditionFalse, status)
		assert.True(t, found)

		noConditionNode := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node3"}}
		status, found = getNodeReadyConditionFromNode(noConditionNode)
		assert.Equal(t, corev1.ConditionStatus(""), status)
		assert.False(t, found)
	})
}

func TestNodeManagement(t *testing.T) {
	t.Run("AddOrUpdateNodeLabel", func(t *testing.T) {
		tests := []struct {
			name          string
			node          *corev1.Node
			nodeGroupName string
			shouldError   bool
			errorType     string
		}{
			{
				name:          "add label to node",
				node:          createNode("node1", nil, true),
				nodeGroupName: "group1",
				shouldError:   false,
			},
			{
				name:          "node belongs to same nodegroup",
				node:          createNode("node1", map[string]string{LabelBelongingTo: "group1"}, true),
				nodeGroupName: "group1",
				shouldError:   false,
			},
			{
				name:          "node belongs to different nodegroup",
				node:          createNode("node1", map[string]string{LabelBelongingTo: "group2"}, true),
				nodeGroupName: "group1",
				shouldError:   true,
			},
			{
				name:          "patch error",
				node:          createNode("node1", nil, true),
				nodeGroupName: "group1",
				shouldError:   true,
				errorType:     "patch",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				client := NewMockClient(tt.node)
				if tt.errorType != "" {
					client.shouldErrorOn[tt.errorType] = true
				}

				controller := &Controller{Client: client}
				err := controller.addOrUpdateNodeLabel(context.Background(), tt.node, tt.nodeGroupName)

				if tt.shouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)

					updatedNode := &corev1.Node{}
					_ = client.Get(context.Background(), types.NamespacedName{Name: tt.node.Name}, updatedNode)
					if !tt.shouldError {
						assert.Equal(t, tt.nodeGroupName, updatedNode.Labels[LabelBelongingTo])
					}
				}
			})
		}
	})

	t.Run("EvictFunctions", func(t *testing.T) {
		node1 := createNode("node1", map[string]string{LabelBelongingTo: "group1"}, true)
		node2 := createNode("node2", map[string]string{LabelBelongingTo: "group1"}, true)
		node3 := createNode("node3", map[string]string{LabelBelongingTo: "group2"}, true)
		pod1 := createPod("pod1", "default", "node1", map[string]string{LabelBelongingTo: "group1"})
		pod2 := createPod("pod2", "default", "node1", map[string]string{"app": "web"})

		t.Run("EvictPodsShouldNotRunOnNode", func(t *testing.T) {
			client := NewMockClient(node1, pod1, pod2)
			controller := &Controller{Client: client}

			err := controller.evictPodsShouldNotRunOnNode(context.Background(), node1, "group1")
			assert.NoError(t, err)

			errorCases := []struct {
				name      string
				errorType string
			}{
				{"list error", "list"},
				{"delete error", "delete"},
			}

			for _, ec := range errorCases {
				t.Run(ec.name, func(t *testing.T) {
					client := NewMockClient(node1, pod1, pod2)
					client.shouldErrorOn[ec.errorType] = true
					controller := &Controller{Client: client}

					err := controller.evictPodsShouldNotRunOnNode(context.Background(), node1, "group1")
					assert.Error(t, err)
				})
			}
		})

		t.Run("EvictNodes", func(t *testing.T) {
			client := NewMockClient(node1, node2, pod1, pod2)
			controller := &Controller{Client: client}

			err := controller.evictNodes(context.Background(), []corev1.Node{*node1, *node2})
			assert.NoError(t, err)

			updatedNode := &corev1.Node{}
			_ = client.Get(context.Background(), types.NamespacedName{Name: "node1"}, updatedNode)
			assert.NotContains(t, updatedNode.Labels, LabelBelongingTo)

			client = NewMockClient(node1, node2, pod1, pod2)
			client.shouldErrorOn["patch"] = true
			controller = &Controller{Client: client}

			err = controller.evictNodes(context.Background(), []corev1.Node{*node1, *node2})
			assert.Error(t, err)
		})

		t.Run("EvictNodesInNodegroup", func(t *testing.T) {
			client := NewMockClient(node1, node2, node3)
			controller := &Controller{Client: client}

			err := controller.evictNodesInNodegroup(context.Background(), "group1")
			assert.NoError(t, err)

			updatedNode1 := &corev1.Node{}
			_ = client.Get(context.Background(), types.NamespacedName{Name: "node1"}, updatedNode1)
			assert.NotContains(t, updatedNode1.Labels, LabelBelongingTo)

			updatedNode3 := &corev1.Node{}
			_ = client.Get(context.Background(), types.NamespacedName{Name: "node3"}, updatedNode3)
			assert.Equal(t, "group2", updatedNode3.Labels[LabelBelongingTo])

			client = NewMockClient(node1, node2, node3)
			client.shouldErrorOn["list"] = true
			controller = &Controller{Client: client}

			err = controller.evictNodesInNodegroup(context.Background(), "group1")
			assert.Error(t, err)
		})
	})

	t.Run("RemoveFinalizer", func(t *testing.T) {
		testCases := []struct {
			name              string
			initialFinalizers []string
			errorType         string
			shouldError       bool
		}{
			{
				name:              "remove existing finalizer",
				initialFinalizers: []string{NodeGroupControllerFinalizer},
				shouldError:       false,
			},
			{
				name:              "no finalizer to remove",
				initialFinalizers: []string{},
				shouldError:       false,
			},
			{
				name:              "update error",
				initialFinalizers: []string{NodeGroupControllerFinalizer},
				errorType:         "update",
				shouldError:       true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				nodeGroup := &appsv1alpha1.NodeGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-group",
						Finalizers: tc.initialFinalizers,
					},
				}

				client := NewMockClient(nodeGroup)
				if tc.errorType != "" {
					client.shouldErrorOn[tc.errorType] = true
				}

				controller := &Controller{Client: client}
				err := controller.removeFinalizer(context.Background(), nodeGroup)

				if tc.shouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)

					updatedNodeGroup := &appsv1alpha1.NodeGroup{}
					_ = client.Get(context.Background(), types.NamespacedName{Name: nodeGroup.Name}, updatedNodeGroup)

					finalizers := updatedNodeGroup.Finalizers
					if finalizers == nil {
						finalizers = []string{}
					}

					assert.NotContains(t, finalizers, NodeGroupControllerFinalizer)
				}
			})
		}
	})
}

func TestNodeSelection(t *testing.T) {
	node1 := createNode("node1", map[string]string{"zone": "east"}, true)
	node2 := createNode("node2", map[string]string{"zone": "west"}, true)
	node3 := createNode("node3", nil, true)

	t.Run("GetNodesByLabels", func(t *testing.T) {
		client := NewMockClient(node1, node2)
		controller := &Controller{Client: client}

		nodes, err := controller.getNodesByLabels(context.Background(), map[string]string{"zone": "east"})
		assert.NoError(t, err)
		assert.Len(t, nodes, 1)
		assert.Equal(t, "node1", nodes[0].Name)

		nodes, err = controller.getNodesByLabels(context.Background(), nil)
		assert.NoError(t, err)
		assert.Empty(t, nodes)

		client.shouldErrorOn["list"] = true
		nodes, err = controller.getNodesByLabels(context.Background(), map[string]string{"zone": "east"})
		assert.Error(t, err)
		assert.Empty(t, nodes)
	})

	t.Run("GetNodesByNodeName", func(t *testing.T) {
		client := NewMockClient(node1, node2)
		controller := &Controller{Client: client}

		nodes, err := controller.getNodesByNodeName(context.Background(), []string{"node1", "node2"})
		assert.NoError(t, err)
		assert.Len(t, nodes, 2)

		client = NewMockClient(node1, node2)
		controller = &Controller{Client: client}

		nodes, err = controller.getNodesByNodeName(context.Background(), []string{"node1", "nonexistent"})
		assert.Error(t, err)
		assert.Len(t, nodes, 1)

		client = NewMockClient(node1, node2)
		client.shouldErrorOn["get"] = true
		controller = &Controller{Client: client}

		nodes, err = controller.getNodesByNodeName(context.Background(), []string{"node1"})
		assert.Error(t, err)
		assert.Empty(t, nodes)
	})

	t.Run("GetNodesSelectedBy", func(t *testing.T) {
		testCases := []struct {
			name      string
			nodeGroup *appsv1alpha1.NodeGroup
			errorType string
			wantLen   int
		}{
			{
				name:      "select by label",
				nodeGroup: createNG("group1", map[string]string{"zone": "east"}, nil),
				wantLen:   1,
			},
			{
				name:      "select by name",
				nodeGroup: createNG("group1", nil, []string{"node3"}),
				wantLen:   1,
			},
			{
				name:      "select by both",
				nodeGroup: createNG("group1", map[string]string{"zone": "east"}, []string{"node3"}),
				wantLen:   2,
			},
			{
				name:      "list error",
				nodeGroup: createNG("group1", map[string]string{"zone": "east"}, nil),
				errorType: "list",
				wantLen:   0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				client := NewMockClient(node1, node2, node3)
				if tc.errorType != "" {
					client.shouldErrorOn[tc.errorType] = true
				}

				controller := &Controller{Client: client}
				nodes, err := controller.getNodesSelectedBy(context.Background(), tc.nodeGroup)

				if tc.errorType != "" {
					assert.Error(t, err)
				}

				assert.Len(t, nodes, tc.wantLen)
			})
		}
	})

	t.Run("NodeMapFunc", func(t *testing.T) {
		nodeGroup1 := createNG("group1", map[string]string{"zone": "east"}, []string{"node3"})
		nodeGroup2 := createNG("group2", map[string]string{"zone": "west"}, []string{})

		testCases := []struct {
			name       string
			node       *corev1.Node
			errorType  string
			wantGroups []string
		}{
			{
				name:       "node with belonging label",
				node:       createNode("node1", map[string]string{LabelBelongingTo: "group1"}, true),
				wantGroups: []string{"group1"},
			},
			{
				name:       "node matching by label",
				node:       createNode("node4", map[string]string{"zone": "east"}, true),
				wantGroups: []string{"group1"},
			},
			{
				name:       "node matching by name",
				node:       createNode("node3", nil, true),
				wantGroups: []string{"group1"},
			},
			{
				name:       "no match",
				node:       createNode("node5", map[string]string{"zone": "north"}, true),
				wantGroups: []string{},
			},
			{
				name:       "list error",
				node:       createNode("node1", nil, true),
				errorType:  "list",
				wantGroups: []string{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				client := NewMockClient(tc.node, nodeGroup1, nodeGroup2)
				if tc.errorType != "" {
					client.shouldErrorOn[tc.errorType] = true
				}

				controller := &Controller{Client: client}
				requests := controller.nodeMapFunc(context.Background(), tc.node)

				requestNames := make([]string, len(requests))
				for i, req := range requests {
					requestNames[i] = req.Name
				}

				assert.ElementsMatch(t, tc.wantGroups, requestNames)
			})
		}
	})
}

type mockTrackerClient struct {
	client.Client
	statusMap map[string][]appsv1alpha1.NodeStatus
}

func (c *mockTrackerClient) Status() client.StatusWriter {
	return &mockTrackerWriter{
		StatusWriter: c.Client.Status(),
		statusMap:    c.statusMap,
	}
}

type mockTrackerWriter struct {
	client.StatusWriter
	statusMap map[string][]appsv1alpha1.NodeStatus
}

func (sw *mockTrackerWriter) Update(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
	if ng, ok := obj.(*appsv1alpha1.NodeGroup); ok {
		sw.statusMap[ng.Name] = ng.Status.NodeStatuses
	}
	return nil
}

func TestControllerCore(t *testing.T) {
	createTrackerClient := func(objects ...runtime.Object) (client.Client, map[string][]appsv1alpha1.NodeStatus) {
		baseClient := fake.NewClientBuilder().
			WithIndex(&corev1.Pod{}, "spec.nodeName", func(obj client.Object) []string {
				pod := obj.(*corev1.Pod)
				return []string{pod.Spec.NodeName}
			}).
			WithRuntimeObjects(objects...).
			Build()

		statusMap := make(map[string][]appsv1alpha1.NodeStatus)
		return &mockTrackerClient{
			Client:    baseClient,
			statusMap: statusMap,
		}, statusMap
	}

	t.Run("SyncNodeGroup", func(t *testing.T) {
		t.Run("basic functionality", func(t *testing.T) {
			nodeGroup := createNG("group1", map[string]string{"zone": "east"}, []string{"node3"})
			node1 := createNode("node1", map[string]string{"zone": "east"}, true)
			node2 := createNode("node2", map[string]string{LabelBelongingTo: "group1"}, true)
			node3 := createNode("node3", nil, false)

			client, statusMap := createTrackerClient(nodeGroup, node1, node2, node3)
			controller := &Controller{Client: client}

			_, err := controller.syncNodeGroup(context.Background(), nodeGroup)
			assert.NoError(t, err)

			updatedNode1 := &corev1.Node{}
			_ = client.Get(context.Background(), types.NamespacedName{Name: "node1"}, updatedNode1)
			assert.Equal(t, "group1", updatedNode1.Labels[LabelBelongingTo])

			updatedNode2 := &corev1.Node{}
			_ = client.Get(context.Background(), types.NamespacedName{Name: "node2"}, updatedNode2)
			assert.NotContains(t, updatedNode2.Labels, LabelBelongingTo)

			statuses := statusMap["group1"]
			assert.Len(t, statuses, 2)

			hasNode1Status := false
			hasNode3Status := false
			for _, status := range statuses {
				if status.NodeName == "node1" {
					hasNode1Status = true
					assert.Equal(t, appsv1alpha1.SucceededSelection, status.SelectionStatus)
					assert.Equal(t, appsv1alpha1.NodeReady, status.ReadyStatus)
				} else if status.NodeName == "node3" {
					hasNode3Status = true
					assert.Equal(t, appsv1alpha1.SucceededSelection, status.SelectionStatus)
					assert.Equal(t, appsv1alpha1.NodeNotReady, status.ReadyStatus)
				}
			}

			assert.True(t, hasNode1Status, "Should have status for node1")
			assert.True(t, hasNode3Status, "Should have status for node3")
		})

		t.Run("non-existent nodes", func(t *testing.T) {
			nodeGroup := createNG("group1", nil, []string{"node1", "nonexistent"})
			node1 := createNode("node1", nil, true)

			client, statusMap := createTrackerClient(nodeGroup, node1)
			controller := &Controller{Client: client}

			_, err := controller.syncNodeGroup(context.Background(), nodeGroup)
			assert.NoError(t, err)

			statuses := statusMap["group1"]
			assert.Len(t, statuses, 2)

			var nonexistentStatus *appsv1alpha1.NodeStatus
			for i, status := range statuses {
				if status.NodeName == "nonexistent" {
					s := statuses[i]
					nonexistentStatus = &s
					break
				}
			}

			assert.NotNil(t, nonexistentStatus, "Should have status for nonexistent node")
			assert.Equal(t, appsv1alpha1.FailedSelection, nonexistentStatus.SelectionStatus)
			assert.Contains(t, nonexistentStatus.SelectionStatusReason, "does not exist")
		})
	})

	t.Run("Reconcile", func(t *testing.T) {
		t.Run("nodegroup not found", func(t *testing.T) {
			client, _ := createTrackerClient()
			controller := &Controller{Client: client}

			result, err := controller.Reconcile(context.Background(), controllerruntime.Request{
				NamespacedName: types.NamespacedName{Name: "nonexistent"},
			})

			assert.False(t, result.Requeue)
			assert.NoError(t, err)
		})

		t.Run("deletion in progress", func(t *testing.T) {
			now := metav1.Now()
			nodeGroup := createNG("group1", nil, nil)
			nodeGroup.DeletionTimestamp = &now
			node := createNode("node1", map[string]string{LabelBelongingTo: "group1"}, true)

			client, _ := createTrackerClient(nodeGroup, node)
			controller := &Controller{Client: client}

			_, _ = controller.Reconcile(context.Background(), controllerruntime.Request{
				NamespacedName: types.NamespacedName{Name: "group1"},
			})

			updatedNode := &corev1.Node{}
			_ = client.Get(context.Background(), types.NamespacedName{Name: "node1"}, updatedNode)
			assert.NotContains(t, updatedNode.Labels, LabelBelongingTo)
		})

		t.Run("add finalizer", func(t *testing.T) {
			nodeGroup := &appsv1alpha1.NodeGroup{
				ObjectMeta: metav1.ObjectMeta{Name: "group1"},
			}

			client, _ := createTrackerClient(nodeGroup)
			controller := &Controller{Client: client}

			_, _ = controller.Reconcile(context.Background(), controllerruntime.Request{
				NamespacedName: types.NamespacedName{Name: "group1"},
			})

			updatedNodeGroup := &appsv1alpha1.NodeGroup{}
			_ = client.Get(context.Background(), types.NamespacedName{Name: "group1"}, updatedNodeGroup)
			assert.Contains(t, updatedNodeGroup.Finalizers, NodeGroupControllerFinalizer)
		})
	})
}

func TestNodesUnion(t *testing.T) {
	cases := map[string]struct {
		list1 []corev1.Node
		list2 []corev1.Node
		want  []corev1.Node
	}{
		"nil-nil": {
			list1: nil,
			list2: nil,
			want:  []corev1.Node{},
		},
		"nil-empty": {
			list1: nil,
			list2: []corev1.Node{},
			want:  []corev1.Node{},
		},
		"nil-normal": {
			list1: nil,
			list2: []corev1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node1",
					},
				},
			},
			want: []corev1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node1",
					},
				},
			},
		},
		"normal-normal-different": {
			list1: []corev1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node1",
					},
				},
			},
			list2: []corev1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node2",
					},
				},
			},
			want: []corev1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node1",
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node2",
					},
				},
			},
		},
		"normal-normal-intersection": {
			list1: []corev1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node1",
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node2",
					},
				},
			},
			list2: []corev1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node2",
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node3",
					},
				},
			},
			want: []corev1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node1",
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node2",
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "node3",
					},
				},
			},
		},
	}
	for n, c := range cases {
		results := nodesUnion(c.list1, c.list2)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Name < results[j].Name
		})
		if !equality.Semantic.DeepEqual(results, c.want) {
			t.Errorf("failed at case: %s, want: %v, got: %v", n, c.want, results)
		}
	}
}
