package client

import (
	"encoding/json"
	"fmt"

	storagev1 "k8s.io/api/storage/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

//CSINodesGetter to get csinode interface
type CSINodesGetter interface {
	CSINodes(namespace string) CSINodesInterface
}

//CSINodesInterface is interface for client nodes
type CSINodesInterface interface {
	Create(*storagev1.CSINode) (*storagev1.CSINode, error)
	Update(*storagev1.CSINode) error
	Delete(name string) error
	Get(name string) (*storagev1.CSINode, error)
}

type csinodes struct {
	namespace string
	send      SendInterface
}

func newCSINodes(namespace string, s SendInterface) *csinodes {
	return &csinodes{
		send:      s,
		namespace: namespace,
	}
}

func (c *csinodes) Create(cm *storagev1.CSINode) (*storagev1.CSINode, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeCSINode, cm.Name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, cm)
	resMsg, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("create csinode failed, err: %v", err)
	}
	content, err := resMsg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to csinode failed, err: %v", err)
	}
	var node *storagev1.CSINode
	err = json.Unmarshal(content, &node)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to csinode failed, err: %v", err)
	}
	return node, nil
}

func (c *csinodes) Update(cm *storagev1.CSINode) error {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeCSINode, cm.Name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.UpdateOperation, cm)
	_, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return fmt.Errorf("update csinode failed, err: %v", err)
	}
	return nil
}

func (c *csinodes) Delete(name string) error {
	return nil
}

func (c *csinodes) Get(name string) (*storagev1.CSINode, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeCSINode, name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("get csinode failed, err: %v", err)
	}

	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to csinode failed, err: %v", err)
	}

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == metamanager.MetaManagerModuleName {
		return handleCSINodeFromMetaDB(content)
	}
	return handleCSINodeFromMetaManager(content)
}

func handleCSINodeFromMetaDB(content []byte) (*storagev1.CSINode, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to csinode list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("csinode length from meta db is %d", len(lists))
	}

	var csinode *storagev1.CSINode
	err = json.Unmarshal([]byte(lists[0]), &csinode)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to csinode from db failed, err: %v", err)
	}
	return csinode, nil
}

func handleCSINodeFromMetaManager(content []byte) (*storagev1.CSINode, error) {
	var csinode *storagev1.CSINode
	err := json.Unmarshal(content, &csinode)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to csinode failed, err: %v", err)
	}
	return csinode, nil
}
