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

package nodegroup

import (
	"context"
	"errors"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
)

var (
	errLabelList = errors.New("simulated label-selector list failure")
	errNodeGet   = errors.New("simulated node get failure")
)

func newSelectionErrorScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := appsv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add NodeGroup scheme: %v", err)
	}
	return s
}

func podFieldIndexer(obj client.Object) []string {
	return []string{obj.(*corev1.Pod).Spec.NodeName}
}

// selectionErrorFixture returns a member node (with belonging, topology, and
// selector labels), a NodeGroup with an observable initial status, and a Pod
// pinned to the member node.
func selectionErrorFixture() (*corev1.Node, *appsv1alpha1.NodeGroup, *corev1.Pod) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "member-node",
			Labels: map[string]string{
				"role":            "edge",
				LabelBelongingTo:  "test-group",
				LabelTopologyZone: "test-group",
			},
		},
	}
	ng := &appsv1alpha1.NodeGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "test-group"},
		Spec:       appsv1alpha1.NodeGroupSpec{MatchLabels: map[string]string{"role": "edge"}},
		Status: appsv1alpha1.NodeGroupStatus{
			NodeStatuses: []appsv1alpha1.NodeStatus{{
				NodeName:        "member-node",
				SelectionStatus: appsv1alpha1.SucceededSelection,
				ReadyStatus:     appsv1alpha1.Unknown,
			}},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "selected-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			NodeName:     "member-node",
			NodeSelector: map[string]string{LabelBelongingTo: "test-group"},
		},
	}
	return node, ng, pod
}

// assertNoMutation verifies that labels, Pod, and NodeGroup status are
// unchanged from the pre-sync snapshots.
func assertNoMutation(
	t *testing.T,
	cli client.Client,
	wantNode *corev1.Node,
	wantNG *appsv1alpha1.NodeGroup,
	podName string,
) {
	t.Helper()

	gotNode := &corev1.Node{}
	if err := cli.Get(context.Background(), client.ObjectKeyFromObject(wantNode), gotNode); err != nil {
		t.Fatalf("failed to get node: %v", err)
	}
	if !equality.Semantic.DeepEqual(wantNode.Labels, gotNode.Labels) {
		t.Errorf("Node labels changed: got %v, want %v", gotNode.Labels, wantNode.Labels)
	}

	gotPod := &corev1.Pod{}
	if err := cli.Get(context.Background(), client.ObjectKey{Name: podName, Namespace: "default"}, gotPod); err != nil {
		t.Errorf("pinned Pod %q was deleted or inaccessible: %v", podName, err)
	}

	gotNG := &appsv1alpha1.NodeGroup{}
	if err := cli.Get(context.Background(), client.ObjectKeyFromObject(wantNG), gotNG); err != nil {
		t.Fatalf("failed to get nodegroup: %v", err)
	}
	if !equality.Semantic.DeepEqual(wantNG.Status, gotNG.Status) {
		t.Errorf("NodeGroup status changed: got %v, want %v", gotNG.Status, wantNG.Status)
	}
}

// buildFatalErrorClient creates a fake client with the Pod field index and
// an interceptor that fails Node List calls for the MatchLabels selector.
func buildFatalErrorClient(t *testing.T, listErr error, node *corev1.Node, ng *appsv1alpha1.NodeGroup, pod *corev1.Pod) client.Client {
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(newSelectionErrorScheme(t)).
		WithObjects(node, ng, pod).
		WithStatusSubresource(ng).
		WithIndex(&corev1.Pod{}, "spec.nodeName", podFieldIndexer).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*corev1.NodeList); ok {
					lo := &client.ListOptions{}
					for _, o := range opts {
						o.ApplyToList(lo)
					}
					if lo.LabelSelector != nil && lo.LabelSelector.String() != LabelBelongingTo+"=test-group" {
						return listErr
					}
				}
				return c.List(ctx, list, opts...)
			},
		}).
		Build()
}

func TestSyncNodeGroupDoesNotMutateOnSelectionError(t *testing.T) {
	node, ng, pod := selectionErrorFixture()
	cli := buildFatalErrorClient(t, errLabelList, node, ng, pod)

	wantNode := node.DeepCopy()
	wantNG := ng.DeepCopy()

	result, err := NewController(cli).syncNodeGroup(context.Background(), ng)

	if !errors.Is(err, errLabelList) {
		t.Fatalf("expected label-list error, got %v", err)
	}
	if !result.Requeue {
		t.Errorf("expected Requeue=true, got %v", result)
	}
	assertNoMutation(t, cli, wantNode, wantNG, "selected-pod")
}

func TestSyncNodeGroupAbortsOnNotFoundFromLabelList(t *testing.T) {
	node, ng, pod := selectionErrorFixture()

	labelListNotFound := apierrors.NewNotFound(
		schema.GroupResource{Resource: "nodes"}, "",
	)
	cli := buildFatalErrorClient(t, labelListNotFound, node, ng, pod)

	wantNode := node.DeepCopy()
	wantNG := ng.DeepCopy()

	result, err := NewController(cli).syncNodeGroup(context.Background(), ng)

	if !errors.Is(err, labelListNotFound) {
		t.Fatalf("expected label-list NotFound error, got %v", err)
	}
	if !result.Requeue {
		t.Errorf("expected Requeue=true, got %v", result)
	}
	assertNoMutation(t, cli, wantNode, wantNG, "selected-pod")
}

func TestSyncNodeGroupToleratesNotFoundForNamedNodes(t *testing.T) {
	existingNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "existing-node",
			Labels: map[string]string{LabelBelongingTo: "test-group"},
		},
	}
	ng := &appsv1alpha1.NodeGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "test-group"},
		Spec:       appsv1alpha1.NodeGroupSpec{Nodes: []string{"existing-node", "missing-node"}},
	}

	cli := fake.NewClientBuilder().
		WithScheme(newSelectionErrorScheme(t)).
		WithObjects(existingNode, ng).
		WithStatusSubresource(ng).
		WithIndex(&corev1.Pod{}, "spec.nodeName", podFieldIndexer).
		Build()

	result, err := NewController(cli).syncNodeGroup(context.Background(), ng)

	if err != nil {
		t.Fatalf("expected no error when only NotFound nodes are missing, got: %v", err)
	}
	if result.Requeue {
		t.Errorf("expected Requeue=false, got %v", result)
	}

	gotNode := &corev1.Node{}
	if err := cli.Get(context.Background(), client.ObjectKeyFromObject(existingNode), gotNode); err != nil {
		t.Fatalf("failed to get node: %v", err)
	}
	if _, ok := gotNode.Labels[LabelBelongingTo]; !ok {
		t.Errorf("membership label was removed from existing node")
	}

	gotNG := &appsv1alpha1.NodeGroup{}
	if err := cli.Get(context.Background(), client.ObjectKeyFromObject(ng), gotNG); err != nil {
		t.Fatalf("failed to get nodegroup: %v", err)
	}
	var ms *appsv1alpha1.NodeStatus
	for i := range gotNG.Status.NodeStatuses {
		if gotNG.Status.NodeStatuses[i].NodeName == "missing-node" {
			ms = &gotNG.Status.NodeStatuses[i]
			break
		}
	}
	if ms == nil {
		t.Fatal("expected status entry for missing-node, found none")
	}
	if ms.SelectionStatus != appsv1alpha1.FailedSelection {
		t.Errorf("expected SelectionStatus=%q, got %q", appsv1alpha1.FailedSelection, ms.SelectionStatus)
	}
	if ms.SelectionStatusReason != "node does not exist" {
		t.Errorf("expected SelectionStatusReason=%q, got %q", "node does not exist", ms.SelectionStatusReason)
	}
	if ms.ReadyStatus != appsv1alpha1.Unknown {
		t.Errorf("expected ReadyStatus=%q, got %q", appsv1alpha1.Unknown, ms.ReadyStatus)
	}
}

func TestSyncNodeGroupAbortsOnNonNotFoundNodeGetError(t *testing.T) {
	node, ng, pod := selectionErrorFixture()
	ng.Spec = appsv1alpha1.NodeGroupSpec{Nodes: []string{"member-node", "error-node"}}

	cli := fake.NewClientBuilder().
		WithScheme(newSelectionErrorScheme(t)).
		WithObjects(node, ng, pod).
		WithStatusSubresource(ng).
		WithIndex(&corev1.Pod{}, "spec.nodeName", podFieldIndexer).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Node); ok && key.Name == "error-node" {
					return fmt.Errorf("get node error-node: %w", errNodeGet)
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).
		Build()

	wantNode := node.DeepCopy()
	wantNG := ng.DeepCopy()

	result, err := NewController(cli).syncNodeGroup(context.Background(), ng)

	if !errors.Is(err, errNodeGet) {
		t.Fatalf("expected node-get error, got %v", err)
	}
	if !result.Requeue {
		t.Errorf("expected Requeue=true, got %v", result)
	}
	assertNoMutation(t, cli, wantNode, wantNG, "selected-pod")
}
