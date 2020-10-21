package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// FakePersistentVolumeClaims implements PersistentVolumeClaimInterface
type FakeConfigMap struct {
	fakecorev1.FakeConfigMaps
	ns         string
	MetaClient client.CoreInterface
}

// Get takes name of the persistentVolumeClaim, and returns the corresponding persistentVolumeClaim object
func (c *FakeConfigMap) Get(ctx context.Context, name string, options metav1.GetOptions) (result *corev1.ConfigMap, err error) {
	return c.MetaClient.ConfigMaps(c.ns).Get(name)
}
