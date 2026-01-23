package client

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

// NodesGetter to get node interface
type NodesGetter interface {
	Nodes(namespace string) NodesInterface
}

// NodesInterface is interface for client nodes
type NodesInterface interface {
	Create(*api.Node) (*api.Node, error)
	Update(*api.Node) error
	Patch(name string, patchBytes []byte) (*api.Node, error)
	Delete(name string) error
	Get(name string) (*api.Node, error)
}

type nodes struct {
	namespace string
	send      SendInterface
}

// NodeResp represents node response from the api-server
type NodeResp struct {
	Object *api.Node
	Err    apierrors.StatusError
}

func newNodes(namespace string, s SendInterface) *nodes {
	return &nodes{
		send:      s,
		namespace: namespace,
	}
}

func (c *nodes) Create(cm *api.Node) (*api.Node, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNode, cm.Name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, cm)
	resp, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("create node failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to node failed, err: %v", err)
	}

	return handleNodeResp(resource, content)
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

func (c *nodes) Patch(name string, data []byte) (*api.Node, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNodePatch, name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.PatchOperation, string(data))
	resp, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("update node failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to node failed, err: %v", err)
	}

	return handleNodeResp(resource, content)
}

func (c *nodes) Delete(string) error {
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

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == modules.MetaManagerModuleName {
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

func handleNodeResp(resource string, content []byte) (*api.Node, error) {
	var nodeResp NodeResp
	err := json.Unmarshal(content, &nodeResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node failed, err: %v", err)
	}

	if reflect.DeepEqual(nodeResp.Err, apierrors.StatusError{}) {
		if err = updateNodeDB(resource, nodeResp.Object); err != nil {
			return nil, fmt.Errorf("update node meta failed, err: %v", err)
		}
		return nodeResp.Object, nil
	}
	return nodeResp.Object, &nodeResp.Err
}

func updateNodeDB(resource string, node *api.Node) error {
	node.APIVersion = "v1"
	node.Kind = "Node"
	nodeContent, err := json.Marshal(node)
	if err != nil {
		klog.Errorf("unmarshal resp node failed, err: %v", err)
		return err
	}
	nodeKey := strings.Replace(resource,
		constants.ResourceSep+model.ResourceTypeNodePatch+constants.ResourceSep,
		constants.ResourceSep+model.ResourceTypeNode+constants.ResourceSep, 1)

	meta := &dao.Meta{
		Key:   nodeKey,
		Type:  model.ResourceTypeNode,
		Value: string(nodeContent)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		return err
	}
	return nil
}
