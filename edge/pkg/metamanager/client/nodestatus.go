package client

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
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
	send      SendInterface
}

func newNodeStatus(namespace string, s SendInterface) *nodeStatus {
	return &nodeStatus{
		send:      s,
		namespace: namespace,
	}
}

func (c *nodeStatus) Create(ns *edgeapi.NodeStatusRequest) (*edgeapi.NodeStatusRequest, error) {
	return nil, nil
}

func (c *nodeStatus) Update(rsName string, ns edgeapi.NodeStatusRequest) error {
	// node status is periodic reporting to the cloud, if edge and cloud connection
	// is interrupted, we just return error to prevent a large number of messages
	// accumulating in the channel, and mass message hits cloudCore when network
	// connection is established between edge and cloud.
	if !connect.IsConnected() {
		return fmt.Errorf("edge and cloud connection is interrupted")
	}

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
