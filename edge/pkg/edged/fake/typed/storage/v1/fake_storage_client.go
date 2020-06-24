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
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

type FakeStorageV1 struct {
	fakestoragev1.FakeStorageV1
	MetaClient client.CoreInterface
}

func (c *FakeStorageV1) VolumeAttachments() storagev1.VolumeAttachmentInterface {
	return &FakeVolumeAttachments{fakestoragev1.FakeVolumeAttachments{Fake: &c.FakeStorageV1}, c.MetaClient}
}
