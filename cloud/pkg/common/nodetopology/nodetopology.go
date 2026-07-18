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

// Package nodetopology determines whether kube-apiserver is reachable via a
// node-local iptables DNAT rule installed on a given node, i.e. whether that
// node and kube-apiserver are "colocated". It is shared by every CloudCore
// module that needs to gate EdgeTunnelIP on real cluster topology instead of
// static configuration, so there is exactly one implementation of the check.
package nodetopology

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
)

// NodeNameEnvVar is the downward-API environment variable a component's
// deployment must inject (fieldRef: spec.nodeName) so a running instance can
// identify which node it is scheduled on.
const NodeNameEnvVar = "NODE_NAME"

// IsAPIServerColocated reports whether every reachable kube-apiserver
// endpoint resolves to an address of nodeName, i.e. whether a node-local
// iptables DNAT rule installed on that node would actually intercept the API
// server's outbound traffic.
//
// The real, routable API server address(es) are read from the well-known
// default/kubernetes Endpoints object rather than by inspecting Node or Pod
// objects: that Endpoints object is populated by the API server itself, so
// it is accurate for both self-hosted (kubeadm/static-pod) control planes
// and managed control planes (EKS/GKE/AKS/...) where the API server is not a
// node in the cluster at all.
//
// determined reports whether a definitive answer was possible. Callers must
// not treat (false, false) as "not colocated" -- it means unknown, e.g.
// nodeName is empty, or the lookups failed.
func IsAPIServerColocated(ctx context.Context, kubeClient v1.CoreV1Interface, nodeName string) (sameNode bool, determined bool) {
	if kubeClient == nil || nodeName == "" {
		return false, false
	}

	node, err := kubeClient.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		klog.V(4).Infof("failed to get own node %q for EdgeTunnelIP placement check: %v", nodeName, err)
		return false, false
	}
	nodeIPs := make(map[string]bool, len(node.Status.Addresses))
	for _, addr := range node.Status.Addresses {
		nodeIPs[addr.Address] = true
	}

	endpoints, err := kubeClient.Endpoints(corev1.NamespaceDefault).Get(ctx, "kubernetes", metav1.GetOptions{})
	if err != nil {
		klog.V(4).Infof("failed to get default/kubernetes endpoints for EdgeTunnelIP placement check: %v", err)
		return false, false
	}

	found := false
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			found = true
			if !nodeIPs[addr.IP] {
				// This API server replica is not reachable via nodeName --
				// a DNAT rule installed there cannot intercept it.
				return false, true
			}
		}
	}
	return found, found
}
