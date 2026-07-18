/*
Copyright 2026 The KubeEdge Authors.

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

package nodetopology

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func kubernetesEndpoints(ips ...string) *corev1.Endpoints {
	addrs := make([]corev1.EndpointAddress, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, corev1.EndpointAddress{IP: ip})
	}
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "kubernetes", Namespace: corev1.NamespaceDefault},
		Subsets:    []corev1.EndpointSubset{{Addresses: addrs}},
	}
}

func cloudCoreNode(name string, ips ...string) *corev1.Node {
	addrs := make([]corev1.NodeAddress, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: ip})
	}
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     corev1.NodeStatus{Addresses: addrs},
	}
}

// TestIsAPIServerColocated covers the HA case in particular: even one
// apiserver replica outside nodeName means that replica's traffic needs
// EdgeTunnelIP, so it must not count as colocated.
func TestIsAPIServerColocated(t *testing.T) {
	const cloudCoreNodeName = "cloudcore-node"

	t.Run("OwnNodeNotFound", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(kubernetesEndpoints("10.0.0.9"))
		same, determined := IsAPIServerColocated(context.Background(), fakeClient.CoreV1(), cloudCoreNodeName)
		assert.False(t, same)
		assert.False(t, determined)
	})

	t.Run("EndpointsNotFound", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(cloudCoreNode(cloudCoreNodeName, "10.0.0.9"))
		same, determined := IsAPIServerColocated(context.Background(), fakeClient.CoreV1(), cloudCoreNodeName)
		assert.False(t, same)
		assert.False(t, determined)
	})

	t.Run("HAPartialMatchIsNotColocated", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			cloudCoreNode(cloudCoreNodeName, "10.0.0.9"),
			kubernetesEndpoints("10.0.0.9", "10.0.0.2"), // second replica elsewhere
		)
		same, determined := IsAPIServerColocated(context.Background(), fakeClient.CoreV1(), cloudCoreNodeName)
		assert.False(t, same)
		assert.True(t, determined)
	})

	t.Run("HAFullMatchIsColocated", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			cloudCoreNode(cloudCoreNodeName, "10.0.0.9", "10.0.0.2"),
			kubernetesEndpoints("10.0.0.9", "10.0.0.2"),
		)
		same, determined := IsAPIServerColocated(context.Background(), fakeClient.CoreV1(), cloudCoreNodeName)
		assert.True(t, same)
		assert.True(t, determined)
	})

	t.Run("EmptyNodeName", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(kubernetesEndpoints("10.0.0.9"))
		same, determined := IsAPIServerColocated(context.Background(), fakeClient.CoreV1(), "")
		assert.False(t, same)
		assert.False(t, determined)
	})
}
