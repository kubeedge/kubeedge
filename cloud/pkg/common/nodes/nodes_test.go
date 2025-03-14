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

package nodes

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestIsEdgeNode(t *testing.T) {
	cases := []struct {
		name   string
		labels map[string]string
		want   bool
	}{
		{name: "labels is nil", labels: nil, want: false},
		{name: "labels is empty", labels: map[string]string{}, want: false},
		{
			name: "invalid label value",
			labels: map[string]string{
				constants.EdgeNodeRoleKey: "abc",
			},
			want: false,
		},
		{
			name: "is edge node label",
			labels: map[string]string{
				constants.EdgeNodeRoleKey: constants.EdgeNodeRoleValue,
			},
			want: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := IsEdgeNode(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "node",
					Labels: c.labels,
				},
			})
			assert.Equal(t, c.want, res)
		})
	}
}

func TestIsReadyNode(t *testing.T) {
	cases := []struct {
		name string
		node *corev1.Node
		want bool
	}{
		{
			name: "node is nil",
			node: nil,
			want: false,
		},
		{
			name: "node status conditions is empty",
			node: &corev1.Node{},
			want: false,
		},
		{
			name: "node not ready",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "node is ready",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, IsReadyNode(c.node))
		})
	}
}

func TestGetManagedEdgeNodes(t *testing.T) {
	var nodeSessions sync.Map
	nodeSessions.Store("node1", struct{}{})
	nodeSessions.Store("node2", struct{}{})
	res := GetManagedEdgeNodes(&nodeSessions)
	assert.Len(t, res, 2)
}
