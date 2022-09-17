package client

import (
	"encoding/json"
	"fmt"

	api "k8s.io/api/core/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

//NodesGetter to get node interface
type NodesGetter interface {
	Nodes(namespace string) NodesInterface
}

//NodesInterface is interface for client nodes
type NodesInterface interface {
	Create(*api.Node) (*api.Node, error)
	Update(*api.Node) error
	Delete(name string) error
	Get(name string) (*api.Node, error)
}

type nodes struct {
	namespace string
	send      SendInterface
}

func newNodes(namespace string, s SendInterface) *nodes {
	return &nodes{
		send:      s,
		namespace: namespace,
	}
}

func (c *nodes) Create(cm *api.Node) (*api.Node, error) {
	return nil, nil
}

func (c *nodes) Update(cm *api.Node) error {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNode, cm.Name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.UpdateOperation, cm)
	_, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return fmt.Errorf("update node failed, err: %v", err)
	}
	return nil
}

func (c *nodes) Delete(name string) error {
	return nil
}

func (c *nodes) Get(name string) (*api.Node, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNode, name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("get node failed, err: %v", err)
	}

	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to node failed, err: %v", err)
	}

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == metamanager.MetaManagerModuleName {
		return handleNodeFromMetaDB(content)
	}
	return handleNodeFromMetaManager(content)
}

func handleNodeFromMetaDB(content []byte) (*api.Node, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("node length from meta db is %d", len(lists))
	}

	var node api.Node
	err = json.Unmarshal([]byte(lists[0]), &node)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node from db failed, err: %v", err)
	}
	return &node, nil
}

func handleNodeFromMetaManager(content []byte) (*api.Node, error) {
	var node api.Node
	err := json.Unmarshal(content, &node)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node failed, err: %v", err)
	}
	return &node, nil
}
