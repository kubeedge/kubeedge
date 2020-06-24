/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"fmt"

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
