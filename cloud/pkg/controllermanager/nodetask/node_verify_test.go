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

package nodetask

import (
	"context"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestVerifyNodeDefine(t *testing.T) {
	ctx := context.TODO()
	t.Run("nodeNames and nodeSelector conflicts", func(t *testing.T) {
		_, err := VerifyNodeDefine(ctx, nil, []string{"node1", "node2"}, &metav1.LabelSelector{})
		assert.Error(t, err)
		assert.ErrorContains(t, err, "nodeNames and nodeSelector cannot be specified at the same time")
	})

	t.Run("nodeNames and nodeSelectors are empty", func(t *testing.T) {
		_, err := VerifyNodeDefine(ctx, nil, []string{}, nil)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "nodeNames and nodeSelector cannot both be empty")
	})

	t.Run("no node matched", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(verifyNodeBySelector,
			func(_ctx context.Context, _che cache.Cache, _nodeSelector *metav1.LabelSelector,
			) ([]NodeVerificationResult, error) {
				return []NodeVerificationResult{}, nil
			})

		_, err := VerifyNodeDefine(ctx, nil, []string{}, &metav1.LabelSelector{
			MatchLabels: map[string]string{"key": "value"},
		})
		assert.Error(t, err)
		assert.ErrorContains(t, err, "no nodes were matched with selector")
	})

	t.Run("verify successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(verifyNodeByNames,
			func(_ctx context.Context, _che cache.Cache, names []string) []NodeVerificationResult {
				res := make([]NodeVerificationResult, 0, len(names))
				for _, name := range names {
					res = append(res, NodeVerificationResult{
						NodeName: name,
					})
				}
				return res
			})
		res, err := VerifyNodeDefine(ctx, nil, []string{"node1", "node2"}, nil)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})
}

func TestVerifyNodeByNames(t *testing.T) {
	ctx := context.TODO()
	cheimpl := informertest.FakeInformers{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyMethodFunc(&cheimpl, "Get",
		func(_ctx context.Context, key client.ObjectKey, obj client.Object, _opts ...client.GetOption) error {
			node, ok := obj.(*corev1.Node)
			if !ok {
				return errors.New("invalid object type")
			}
			switch key.Name {
			case "node1": // Effective edge node.
				node.Name = key.Name
				node.Labels = map[string]string{constants.EdgeNodeRoleKey: ""}
				node.Status = corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				}
			case "node2": // Node is not ready.
				node.Name = key.Name
				node.Labels = map[string]string{constants.EdgeNodeRoleKey: ""}
				node.Status = corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						},
					},
				}
			case "node3": // Node is not edge node.
				node.Name = key.Name
				node.Status = corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				}
			default:
				return errors.New("not found")
			}
			return nil
		})

	res := verifyNodeByNames(ctx, &cheimpl, []string{"node1", "node2", "node3", "node4"})
	assert.Len(t, res, 4)
	assert.Empty(t, res[0].ErrorMessage)
	assert.Equal(t, "the node is not ready", res[1].ErrorMessage)
	assert.Equal(t, "the node is not an edge node", res[2].ErrorMessage)
	assert.Contains(t, res[3].ErrorMessage, "failed to get node")
}

func TestVerifyNodeBySelector(t *testing.T) {
	ctx := context.TODO()
	cheimpl := informertest.FakeInformers{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyMethodFunc(&cheimpl, "List",
		func(_ctx context.Context, list client.ObjectList, _opts ...client.ListOption) error {
			nodes, ok := list.(*corev1.NodeList)
			if !ok {
				return errors.New("invalid object type")
			}
			nodes.Items = []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "node1",
						Labels: map[string]string{constants.EdgeNodeRoleKey: ""},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				{ // Node is not ready.
					ObjectMeta: metav1.ObjectMeta{
						Name:   "node2",
						Labels: map[string]string{constants.EdgeNodeRoleKey: ""},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
				{ // Node is not edge node.
					ObjectMeta: metav1.ObjectMeta{
						Name: "node3",
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			}
			return nil
		})

	res, err := verifyNodeBySelector(ctx, &cheimpl, &metav1.LabelSelector{})
	assert.NoError(t, err)
	assert.Len(t, res, 3)
	assert.Empty(t, res[0].ErrorMessage)
	assert.Equal(t, "the node is not ready", res[1].ErrorMessage)
	assert.Equal(t, "the node is not an edge node", res[2].ErrorMessage)
}
