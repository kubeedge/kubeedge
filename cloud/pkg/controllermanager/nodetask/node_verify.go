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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/nodes"
)

// NodeVerificationResult is the result of node verification.
type NodeVerificationResult struct {
	NodeName string
	// ErrorMessage is the error message of node verification.
	// If it is empty, it means that the node verification is successful.
	ErrorMessage string
}

// VerifyNodeDefine verifies the node definition of node task.
func VerifyNodeDefine(
	ctx context.Context,
	che cache.Cache,
	nodeNames []string,
	nodeSelector *metav1.LabelSelector,
) (res []NodeVerificationResult, err error) {
	fmt.Println("--------- nodeNames empty?", len(nodeNames), ", nodeSelector nil?", nodeSelector == nil)
	if len(nodeNames) > 0 && nodeSelector != nil {
		return nil, errors.New("nodeNames and nodeSelector cannot be specified at the same time")
	}
	if len(nodeNames) > 0 {
		res = verifyNodeByNames(ctx, che, nodeNames)
	} else if nodeSelector != nil {
		res, err = verifyNodeBySelector(ctx, che, nodeSelector)
		if err != nil {
			return
		}
	} else {
		return nil, errors.New("nodeNames and nodeSelector cannot both be empty")
	}
	if len(res) == 0 {
		err = errors.New("no nodes were matched with selector")
		return
	}
	return
}

func verifyNodeByNames(ctx context.Context, che cache.Cache, names []string) []NodeVerificationResult {
	res := make([]NodeVerificationResult, 0)
	for _, name := range names {
		var (
			node   corev1.Node
			errmsg string
		)
		if err := che.Get(ctx, client.ObjectKey{Name: name}, &node); err != nil {
			errmsg = fmt.Sprintf("failed to get node, err: %v", err)
		} else if !nodes.IsEdgeNode(&node) {
			errmsg = "the node is not an edge node"
		} else if !nodes.IsReadyNode(&node) {
			errmsg = "the node is not ready"
		}
		res = append(res, NodeVerificationResult{
			NodeName:     name,
			ErrorMessage: errmsg,
		})
	}
	return res
}

func verifyNodeBySelector(ctx context.Context, che cache.Cache, nodeSelector *metav1.LabelSelector,
) ([]NodeVerificationResult, error) {
	var nodeList corev1.NodeList
	selector, err := metav1.LabelSelectorAsSelector(nodeSelector)
	if err != nil {
		return nil, fmt.Errorf("nodeSelector is invalid, err: %v", err)
	}
	if err := che.List(ctx, &nodeList, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return nil, fmt.Errorf("failed to list nodes with selector, err: %v", err)
	}

	res := make([]NodeVerificationResult, 0)
	for i := range nodeList.Items {
		var errmsg string
		node := nodeList.Items[i]
		if !nodes.IsEdgeNode(&node) {
			errmsg = "the node is not an edge node"
		} else if !nodes.IsReadyNode(&node) {
			errmsg = "the node is not ready"
		}
		res = append(res, NodeVerificationResult{
			NodeName:     node.Name,
			ErrorMessage: errmsg,
		})
	}
	return res, nil
}
