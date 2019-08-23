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
