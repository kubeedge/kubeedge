/*
Copyright 2015 The Kubernetes Authors.

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

package pod

import (
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// MirrorClient knows how to create/delete a mirror pod in the API server.
type MirrorClient interface {
}

// nodeGetter is a subset of NodeLister, simplified for testing.
type nodeGetter interface {
	// Get retrieves the Node for a given name.
	Get(name string) (*v1.Node, error)
}

// basicMirrorClient is a functional MirrorClient.  Mirror pods are stored in
// the kubelet directly because they need to be in sync with the internal
// pods.
type basicMirrorClient struct {
	apiserverClient clientset.Interface
	nodeGetter      nodeGetter
	nodeName        string
}

// NewBasicMirrorClient returns a new MirrorClient.
func NewBasicMirrorClient(apiserverClient clientset.Interface, nodeName string, nodeGetter nodeGetter) MirrorClient {
	return &basicMirrorClient{
		apiserverClient: apiserverClient,
		nodeName:        nodeName,
		nodeGetter:      nodeGetter,
	}
}
