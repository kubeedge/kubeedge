package client

import (
	"encoding/json"
	"fmt"

	api "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

// VolumeAttachmentsGetter is interface to get client VolumeAttachments
type VolumeAttachmentsGetter interface {
	VolumeAttachments(namespace string) VolumeAttachmentsInterface
}

// VolumeAttachmentsInterface is interface for client VolumeAttachments
type VolumeAttachmentsInterface interface {
	Create(*api.VolumeAttachment) (*api.VolumeAttachment, error)
	Update(*api.VolumeAttachment) error
	Delete(name string) error
	Get(name string, options metav1.GetOptions) (*api.VolumeAttachment, error)
}

type volumeattachments struct {
	namespace string
	send      SendInterface
}

func newVolumeAttachments(n string, s SendInterface) *volumeattachments {
	return &volumeattachments{
		namespace: n,
		send:      s,
	}
}

func (c *volumeattachments) Create(va *api.VolumeAttachment) (*api.VolumeAttachment, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, "volumeattachment", va.Name)
	vaMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, va)
	_, err := c.send.SendSync(vaMsg)
	if err != nil {
		return nil, fmt.Errorf("create VolumeAttachment failed, err: %v", err)
	}
	return nil, nil
}

func (c *volumeattachments) Update(va *api.VolumeAttachment) error {
	return nil
}

func (c *volumeattachments) Delete(name string) error {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, "volumeattachment", name)
	vaMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.DeleteOperation, nil)
	_, err := c.send.SendSync(vaMsg)
	if err != nil {
		return fmt.Errorf("delete VolumeAttachment failed, err: %v", err)
	}
	return nil
}

func (c *volumeattachments) Get(name string, options metav1.GetOptions) (*api.VolumeAttachment, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, "volumeattachment", name)
	vaMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(vaMsg)
	if err != nil {
		return nil, fmt.Errorf("get volumeattachment from metaManager failed, err: %v", err)
	}

	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to volumeattachment failed, err: %v", err)
	}

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == metamanager.MetaManagerModuleName {
		return handleVolumeAttachmentFromMetaDB(content)
	}
	return handleVolumeAttachmentFromMetaManager(content)
}

func handleVolumeAttachmentFromMetaDB(content []byte) (*api.VolumeAttachment, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to volumeattachment list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("volumeattachment length from meta db is %d", len(lists))
	}

	var va api.VolumeAttachment
	err = json.Unmarshal([]byte(lists[0]), &va)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to volumeattachment from db failed, err: %v", err)
	}
	return &va, nil
}

func handleVolumeAttachmentFromMetaManager(content []byte) (*api.VolumeAttachment, error) {
	var va api.VolumeAttachment
	err := json.Unmarshal(content, &va)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to volumeattachment failed, err: %v", err)
	}
	return &va, nil
}
