package v1

import (
	"context"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakecoordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// LeaseBridge implements LeaseInterface
type LeaseBridge struct {
	fakecoordinationv1.FakeLeases
	ns         string
	MetaClient client.CoreInterface
}

func (c *LeaseBridge)  Create(ctx context.Context, lease *coordinationv1.Lease, opts metav1.CreateOptions) (result *coordinationv1.Lease, err error) {
	return c.MetaClient.Leases(c.ns).Create(lease)
}

func (c *LeaseBridge) Update(ctx context.Context, lease *coordinationv1.Lease, opts metav1.UpdateOptions) (result *coordinationv1.Lease, err error) {
	return c.MetaClient.Leases(c.ns).Update(lease)
}

func (c *LeaseBridge) Get(ctx context.Context, name string, options metav1.GetOptions) (result *coordinationv1.Lease, err error) {
	return c.MetaClient.Leases(c.ns).Get(name)
}
