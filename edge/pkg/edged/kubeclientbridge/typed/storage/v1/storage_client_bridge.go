package v1

import (
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// StorageV1Bridge is a storageV1 bridge
type StorageV1Bridge struct {
	fakestoragev1.FakeStorageV1
	MetaClient client.CoreInterface
}

func (c *StorageV1Bridge) VolumeAttachments() storagev1.VolumeAttachmentInterface {
	return &VolumeAttachmentsBridge{fakestoragev1.FakeVolumeAttachments{Fake: &c.FakeStorageV1}, c.MetaClient}
}
