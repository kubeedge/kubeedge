package testutil

/*
Copyright 2024 The KubeEdge Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	*fake.Clientset
	Corev1 *MockCoreV1
}

func (m *MockClientset) CoreV1() corev1.CoreV1Interface {
	return m.Corev1
}
