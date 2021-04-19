package fake

import (
	clientset "k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"

	kecorev1 "github.com/kubeedge/kubeedge/edge/pkg/edged/fake/typed/core/v1"
	kestoragev1 "github.com/kubeedge/kubeedge/edge/pkg/edged/fake/typed/storage/v1"
)

// NewSimpleClientset is new interface
func NewSimpleClientset() clientset.Interface {
	return &Clientset{*fakekube.NewSimpleClientset()}
}

// Clientset extends Clientset
type Clientset struct {
	fakekube.Clientset
}

// CoreV1 retrieves the CoreV1Client
func (c *Clientset) CoreV1() corev1.CoreV1Interface {
	return &kecorev1.FakeCoreV1{FakeCoreV1: fakecorev1.FakeCoreV1{Fake: &c.Fake}}
}

// StorageV1 retrieves the StorageV1Client
func (c *Clientset) StorageV1() storagev1.StorageV1Interface {
	return &kestoragev1.FakeStorageV1{FakeStorageV1: fakestoragev1.FakeStorageV1{Fake: &c.Fake}}
}
