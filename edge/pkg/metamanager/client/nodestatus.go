package client

import (
	"fmt"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

//NodeStatusGetter is interface to get node status
type NodeStatusGetter interface {
	NodeStatus(namespace string) NodeStatusInterface
}

//NodeStatusInterface is node status interface
type NodeStatusInterface interface {
	Create(*edgeapi.NodeStatusRequest) (*edgeapi.NodeStatusRequest, error)
	Update(rsName string, ns edgeapi.NodeStatusRequest) error
	Delete(name string) error
	Get(name string) (*edgeapi.NodeStatusRequest, error)
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

func (c *nodeStatus) Create(ns *edgeapi.NodeStatusRequest) (*edgeapi.NodeStatusRequest, error) {
	return nil, nil
}

func (c *nodeStatus) Update(rsName string, ns edgeapi.NodeStatusRequest) error {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNodeStatus, rsName)
	nodeStatusMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.UpdateOperation, ns)
	_, err := c.send.SendSync(nodeStatusMsg)
	if err != nil {
		return fmt.Errorf("update nodeStatus failed, err: %v", err)
	}

	return nil
}

func (c *nodeStatus) Delete(name string) error {
	return nil
}

func (c *nodeStatus) Get(name string) (*edgeapi.NodeStatusRequest, error) {
	return nil, nil
}
