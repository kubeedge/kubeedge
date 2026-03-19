package testutil

import (
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type MockCoreV1 struct {
	corev1.CoreV1Interface
	RestClient rest.Interface
}

func (m *MockCoreV1) RESTClient() rest.Interface {
	return m.RestClient
}

type MockClientset struct {
	fake.Clientset
	MockCoreV1 *MockCoreV1
}

func (m *MockClientset) CoreV1() corev1.CoreV1Interface {
	return m.MockCoreV1
}
