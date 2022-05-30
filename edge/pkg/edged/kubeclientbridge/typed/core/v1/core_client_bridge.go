package v1

import (
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// CoreV1Bridge is a coreV1 bridge
type CoreV1Bridge struct {
	fakecorev1.FakeCoreV1
	MetaClient client.CoreInterface
}

func (c *CoreV1Bridge) Nodes() corev1.NodeInterface {
	return &NodesBridge{fakecorev1.FakeNodes{Fake: &c.FakeCoreV1}, c.MetaClient}
}