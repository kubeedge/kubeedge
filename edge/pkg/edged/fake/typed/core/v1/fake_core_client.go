/*
Copyright 2019 The KubeEdge Authors.

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

package v1

import (
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

type FakeCoreV1 struct {
	fakecorev1.FakeCoreV1
	MetaClient client.CoreInterface
}

func (c *FakeCoreV1) Nodes() corev1.NodeInterface {
	return &FakeNodes{fakecorev1.FakeNodes{Fake: &c.FakeCoreV1}, c.MetaClient}
}

func (c *FakeCoreV1) PersistentVolumes() corev1.PersistentVolumeInterface {
	return &FakePersistentVolumes{fakecorev1.FakePersistentVolumes{Fake: &c.FakeCoreV1}, c.MetaClient}
}

func (c *FakeCoreV1) PersistentVolumeClaims(namespace string) corev1.PersistentVolumeClaimInterface {
	return &FakePersistentVolumeClaims{fakecorev1.FakePersistentVolumeClaims{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}

func (c *FakeCoreV1) ConfigMaps(namespace string) corev1.ConfigMapInterface {
	return &FakeConfigMap{fakecorev1.FakeConfigMaps{Fake: &c.FakeCoreV1}, namespace, c.MetaClient}
}
