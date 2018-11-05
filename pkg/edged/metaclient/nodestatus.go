package metaclient

import (
	"fmt"

	"kubeedge/beehive/adoptions/common/api"
	"kubeedge/beehive/pkg/core"
	"kubeedge/beehive/pkg/core/context"
	"kubeedge/beehive/pkg/core/model"
	"kubeedge/pkg/common/message"
)

type NodeStatusGetter interface {
	NodeStatus(namespace string) NodeStatusInterface
}

type NodeStatusInterface interface {
	Create(*api.NodeStatusRequest) (*api.NodeStatusRequest, error)
	Update(rsName string, ns api.NodeStatusRequest) error
	Delete(name string) error
	Get(name string) (*api.NodeStatusRequest, error)
}

type nodeStatus struct {
	namespace string
	context   *context.Context
	send      SendInterface
}

func newNodeStatus(namespace string, c *context.Context, s SendInterface) *nodeStatus {
	return &nodeStatus{
		context:   c,
		send:      s,
		namespace: namespace,
	}
}

func (c *nodeStatus) Create(ns *api.NodeStatusRequest) (*api.NodeStatusRequest, error) {
	return nil, nil
}

func (c *nodeStatus) Update(rsName string, ns api.NodeStatusRequest) error {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNodeStatus, rsName)
	nodeStatusMsg := message.BuildMsg(core.MetaGroup, "", core.EdgedModuleName, resource, model.UpdateOperation, ns)
	_, err := c.send.SendSync(nodeStatusMsg)
	if err != nil {
		return fmt.Errorf("update nodeStatus failed, err: %v", err)
	}

	return nil
}

func (c *nodeStatus) Delete(name string) error {
	return nil
}

func (c *nodeStatus) Get(name string) (*api.NodeStatusRequest, error) {
	return nil, nil
}
