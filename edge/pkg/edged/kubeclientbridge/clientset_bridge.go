/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To make a bridge between kubeclient and metaclient,
This file is derived from K8S client-go code with reduced set of methods
Changes done are
1. Package kubeclientbridge got some functions from "k8s.io/client-go/kubernetes/fake/clientset_generated.go"
and made some variant
*/

package kubeclientbridge

import (
	clientset "k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	fakecoordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"

	kecoordinationv1 "github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge/typed/coordination/v1"
	kecorev1 "github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge/typed/core/v1"
	kestoragev1 "github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge/typed/storage/v1"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// NewSimpleClientset is new interface
func NewSimpleClientset(metaClient client.CoreInterface) clientset.Interface {
	return &Clientset{*fakekube.NewSimpleClientset(), metaClient}
}

// Clientset extends Clientset
type Clientset struct {
	fakekube.Clientset
	MetaClient client.CoreInterface
}

// CoreV1 retrieves the CoreV1Client
func (c *Clientset) CoreV1() corev1.CoreV1Interface {
	return &kecorev1.CoreV1Bridge{FakeCoreV1: fakecorev1.FakeCoreV1{Fake: &c.Fake}, MetaClient: c.MetaClient}
}

// StorageV1 retrieves the StorageV1Client
func (c *Clientset) StorageV1() storagev1.StorageV1Interface {
	return &kestoragev1.StorageV1Bridge{FakeStorageV1: fakestoragev1.FakeStorageV1{Fake: &c.Fake}, MetaClient: c.MetaClient}
}

func (c *Clientset) CoordinationV1() coordinationv1.CoordinationV1Interface {
	return &kecoordinationv1.CoordinationV1Bridge{FakeCoordinationV1: fakecoordinationv1.FakeCoordinationV1{Fake: &c.Fake}, MetaClient: c.MetaClient}
}
