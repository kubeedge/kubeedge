/*
Copyright 2019 The KubeEdge Authors.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// FakeNodes implements NodeInterface
type FakeNodes struct {
	fakecorev1.FakeNodes
	MetaClient client.CoreInterface
}

// Get takes name of the node, and returns the corresponding node object
func (c *FakeNodes) Get(name string, options metav1.GetOptions) (result *corev1.Node, err error) {
	return c.MetaClient.Nodes(metav1.NamespaceDefault).Get(name)
}

// Update takes the representation of a node and updates it
func (c *FakeNodes) Update(node *corev1.Node) (result *corev1.Node, err error) {
	err = c.MetaClient.Nodes(metav1.NamespaceDefault).Update(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}
