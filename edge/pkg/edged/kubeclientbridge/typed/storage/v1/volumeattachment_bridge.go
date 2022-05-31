package v1

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakestoragev1 "k8s.io/client-go/kubernetes/typed/storage/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// VolumeAttachmentsBridge implements PersistentVolumeInterface
type VolumeAttachmentsBridge struct {
	fakestoragev1.FakeVolumeAttachments
	MetaClient client.CoreInterface
}

// Get takes name of the volumeAttachments, and returns the corresponding volumeAttachments object
func (c *VolumeAttachmentsBridge) Get(ctx context.Context, name string, options metav1.GetOptions) (result *storagev1.VolumeAttachment, err error) {
	return c.MetaClient.VolumeAttachments(metav1.NamespaceDefault).Get(name, options)
}
