package nodegroup

import (
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
