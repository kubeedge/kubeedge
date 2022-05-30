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

func (c *CoreV1Bridge) ServiceAccounts(namespace string) corev1.ServiceAccountInterface {
	return &ServiceAccountsBridge{fakecorev1.FakeServiceAccounts{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}

func (c *CoreV1Bridge) ConfigMaps(namespace string) corev1.ConfigMapInterface {
	return &ConfigMapBridge{fakecorev1.FakeConfigMaps{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}

func (c *CoreV1Bridge) Secrets(namespace string) corev1.SecretInterface {
	return &SecretBridge{fakecorev1.FakeSecrets{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}