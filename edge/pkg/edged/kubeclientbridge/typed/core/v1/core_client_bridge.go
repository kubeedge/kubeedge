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
1. Package v1 got some functions from "k8s.io/client-go/kubernetes/typed/core/v1/fake/fake_core_client.go"
and made some variant
*/

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

func (c *CoreV1Bridge) PersistentVolumes() corev1.PersistentVolumeInterface {
	return &PersistentVolumesBridge{fakecorev1.FakePersistentVolumes{Fake: &c.FakeCoreV1}, c.MetaClient}
}

func (c *CoreV1Bridge) PersistentVolumeClaims(namespace string) corev1.PersistentVolumeClaimInterface {
	return &PersistentVolumeClaimsBridge{fakecorev1.FakePersistentVolumeClaims{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}

func (c *CoreV1Bridge) ConfigMaps(namespace string) corev1.ConfigMapInterface {
	return &ConfigMapBridge{fakecorev1.FakeConfigMaps{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}

func (c *CoreV1Bridge) Secrets(namespace string) corev1.SecretInterface {
	return &SecretBridge{fakecorev1.FakeSecrets{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}

func (c *CoreV1Bridge) ServiceAccounts(namespace string) corev1.ServiceAccountInterface {
	return &ServiceAccountsBridge{fakecorev1.FakeServiceAccounts{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}

func (c *CoreV1Bridge) Pods(namespace string) corev1.PodInterface {
	return &PodsBridge{fakecorev1.FakePods{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}
