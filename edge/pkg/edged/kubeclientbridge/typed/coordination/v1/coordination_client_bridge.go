package v1

import (
	v1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	fakecoordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// CoordinationV1Bridge is a coordinationV1 bridge
type CoordinationV1Bridge struct {
	fakecoordinationv1.FakeCoordinationV1
	MetaClient client.CoreInterface
}

func (c *CoordinationV1Bridge) Leases(namespace string) v1.LeaseInterface {
	return &LeaseBridge{fakecoordinationv1.FakeLeases{Fake: &c.FakeCoordinationV1}, namespace, c.MetaClient}
}