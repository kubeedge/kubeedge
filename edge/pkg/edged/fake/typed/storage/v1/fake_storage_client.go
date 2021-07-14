package v1

import (
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"
)

type FakeStorageV1 struct {
	fakestoragev1.FakeStorageV1
}

func (c *FakeStorageV1) VolumeAttachments() storagev1.VolumeAttachmentInterface {
	return &FakeVolumeAttachments{fakestoragev1.FakeVolumeAttachments{Fake: &c.FakeStorageV1}}
}
