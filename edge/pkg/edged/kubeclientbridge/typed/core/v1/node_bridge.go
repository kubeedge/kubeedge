package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// NodesBridge implements NodeInterface
type NodesBridge struct {
	fakecorev1.FakeNodes
	MetaClient client.CoreInterface
}

// Create takes the representation of a node and create it in cluster
func (c *NodesBridge) Create(ctx context.Context, node *corev1.Node, opts metav1.CreateOptions) (*corev1.Node, error) {
	return c.MetaClient.Nodes(metav1.NamespaceDefault).Create(node)
}

// Get takes name of the node, and returns the corresponding node object
func (c *NodesBridge) Get(ctx context.Context, name string, options metav1.GetOptions) (result *corev1.Node, err error) {
	return c.MetaClient.Nodes(metav1.NamespaceDefault).Get(name)
}

// Update takes the representation of a node and updates it
func (c *NodesBridge) Update(ctx context.Context, node *corev1.Node, opts metav1.UpdateOptions) (result *corev1.Node, err error) {
	err = c.MetaClient.Nodes(metav1.NamespaceDefault).Update(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// Patch takes the node patch bytes and updates node status
func (c *NodesBridge) Patch(ctx context.Context, name string, pt kubetypes.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Node, err error) {
	return c.MetaClient.Nodes(metav1.NamespaceDefault).Patch(name, patchBytes)
}
