/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To make a bridge between kubeclient and metaclient,
This file is derived from K8S client-go code with reduced set of methods
Changes done are
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/core/v1/fake/fake_node.go"
and made some variant
*/

package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
)

// NodesBridge implements NodeInterface
type NodesBridge struct {
	fakecorev1.FakeNodes
	MetaClient client.CoreInterface
}

// Create takes the representation of a node and create it in cluster
func (c *NodesBridge) Create(_ context.Context, node *corev1.Node, _ metav1.CreateOptions) (*corev1.Node, error) {
	return c.MetaClient.Nodes(v2.NullNamespace).Create(node)
}

// Get takes name of the node, and returns the corresponding node object
func (c *NodesBridge) Get(_ context.Context, name string, _ metav1.GetOptions) (result *corev1.Node, err error) {
	return c.MetaClient.Nodes(v2.NullNamespace).Get(name)
}

// Update takes the representation of a node and updates it
func (c *NodesBridge) Update(_ context.Context, node *corev1.Node, _ metav1.UpdateOptions) (result *corev1.Node, err error) {
	err = c.MetaClient.Nodes(v2.NullNamespace).Update(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// Patch takes the node patch bytes and updates node status
func (c *NodesBridge) Patch(_ context.Context, name string, _ kubetypes.PatchType, patchBytes []byte, _ metav1.PatchOptions, _ ...string) (result *corev1.Node, err error) {
	return c.MetaClient.Nodes(v2.NullNamespace).Patch(name, patchBytes)
}
