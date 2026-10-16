package nodegroup

import (
	"context"
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
)

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

func TestIfMatchNodeGroup(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: v1.ObjectMeta{
			Name:   "edge-x",
			Labels: map[string]string{"zone": "b"},
		},
	}
	cases := map[string]struct {
		spec appsv1alpha1.NodeGroupSpec
		want bool
	}{
		"selected by name": {
			spec: appsv1alpha1.NodeGroupSpec{Nodes: []string{"edge-1", "edge-x"}},
			want: true,
		},
		"not selected by name, nil matchLabels": {
			spec: appsv1alpha1.NodeGroupSpec{Nodes: []string{"edge-1"}},
			want: false,
		},
		"empty spec": {
			spec: appsv1alpha1.NodeGroupSpec{},
			want: false,
		},
		"selected by labels": {
			spec: appsv1alpha1.NodeGroupSpec{MatchLabels: map[string]string{"zone": "b"}},
			want: true,
		},
		"not selected by labels": {
			spec: appsv1alpha1.NodeGroupSpec{MatchLabels: map[string]string{"zone": "a"}},
			want: false,
		},
		"not selected by name, selected by labels": {
			spec: appsv1alpha1.NodeGroupSpec{
				Nodes:       []string{"edge-1"},
				MatchLabels: map[string]string{"zone": "b"},
			},
			want: true,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			nodegroup := &appsv1alpha1.NodeGroup{
				ObjectMeta: v1.ObjectMeta{Name: "nodegroup"},
				Spec:       c.spec,
			}
			if got := IfMatchNodeGroup(node, nodegroup); got != c.want {
				t.Errorf("IfMatchNodeGroup() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestNodeMapFunc(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := appsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	// a-by-name is listed before b-by-label and only selects nodes by name.
	nodegroups := []runtime.Object{
		&appsv1alpha1.NodeGroup{
			ObjectMeta: v1.ObjectMeta{Name: "a-by-name"},
			Spec:       appsv1alpha1.NodeGroupSpec{Nodes: []string{"edge-1"}},
		},
		&appsv1alpha1.NodeGroup{
			ObjectMeta: v1.ObjectMeta{Name: "b-by-label"},
			Spec:       appsv1alpha1.NodeGroupSpec{MatchLabels: map[string]string{"zone": "b"}},
		},
	}
	c := NewController(fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(nodegroups...).Build())

	request := func(name string) []controllerruntime.Request {
		return []controllerruntime.Request{{NamespacedName: types.NamespacedName{Name: name}}}
	}
	cases := map[string]struct {
		node *corev1.Node
		want []controllerruntime.Request
	}{
		"node with belonging label": {
			node: &corev1.Node{ObjectMeta: v1.ObjectMeta{
				Name:   "edge-y",
				Labels: map[string]string{LabelBelongingTo: "a-by-name"},
			}},
			want: request("a-by-name"),
		},
		"node selected by name": {
			node: &corev1.Node{ObjectMeta: v1.ObjectMeta{Name: "edge-1"}},
			want: request("a-by-name"),
		},
		"node selected by labels": {
			node: &corev1.Node{ObjectMeta: v1.ObjectMeta{
				Name:   "edge-x",
				Labels: map[string]string{"zone": "b"},
			}},
			want: request("b-by-label"),
		},
		"orphan node": {
			node: &corev1.Node{ObjectMeta: v1.ObjectMeta{
				Name:   "edge-z",
				Labels: map[string]string{"zone": "c"},
			}},
			want: []controllerruntime.Request{},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := c.nodeMapFunc(context.TODO(), tc.node)
			if !equality.Semantic.DeepEqual(got, tc.want) {
				t.Errorf("nodeMapFunc() = %v, want %v", got, tc.want)
			}
		})
	}
}
