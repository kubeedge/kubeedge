package v1

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// FakeCSINodes implements CSINodeInterface
type FakeCSINodes struct {
	fakestoragev1.FakeCSINodes
	MetaClient client.CoreInterface
}

// Get takes name of the csinode, and returns the corresponding csinode object
func (c *FakeCSINodes) Get(ctx context.Context, name string, options metav1.GetOptions) (result *storagev1.CSINode, err error) {
	return c.MetaClient.CSINodes(metav1.NamespaceDefault).Get(name)
}

// Get takes name of the csinode, and returns the corresponding csinode object
func (c *FakeCSINodes) Update(ctx context.Context, csinode *storagev1.CSINode, options metav1.UpdateOptions) (result *storagev1.CSINode, err error) {
	err = c.MetaClient.CSINodes(metav1.NamespaceDefault).Update(csinode)
	if err != nil {
		return nil, err
	}
	return csinode, nil
}

// Get takes name of the csinode, and returns the corresponding csinode object
func (c *FakeCSINodes) Create(ctx context.Context, csinode *storagev1.CSINode, options metav1.CreateOptions) (result *storagev1.CSINode, err error) {
	csin, err := c.MetaClient.CSINodes(metav1.NamespaceDefault).Create(csinode)
	if err != nil {
		return nil, err
	}
	return csin, nil
}
