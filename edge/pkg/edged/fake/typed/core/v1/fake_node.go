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
