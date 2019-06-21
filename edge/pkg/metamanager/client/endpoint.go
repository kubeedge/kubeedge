package client

import (
	"encoding/json"
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/constant"

	api "k8s.io/api/core/v1"
)

// EndpointsGetter has a method to return a EndpointsInterface.
// A group's client should implement this interface.
type EndpointsGetter interface {
	Endpoints(namespace string) EndpointsInterface
}

// EndpointsInterface has methods to work with Endpoints resources.
type EndpointsInterface interface {
	Create(*api.Endpoints) (*api.Endpoints, error)
	Update(*api.Endpoints) error
	Delete(name string) error
	Get(name string) (*api.Endpoints, error)
}

// Endpoints is struct implementing EndpointsInterface
type Endpoints struct {
	namespace string
	context   *context.Context
	send      SendInterface
}

func newEndpoints(namespace string, c *context.Context, s SendInterface) *Endpoints {
	return &Endpoints{
		context:   c,
		send:      s,
		namespace: namespace,
	}
}

// Create Endpoints
func (c *Endpoints) Create(cm *api.Endpoints) (*api.Endpoints, error) {
	return nil, nil
}

// Update Endpoints
func (c *Endpoints) Update(cm *api.Endpoints) error {
	return nil
}

// Delete Endpoints
func (c *Endpoints) Delete(name string) error {
	return nil
}

// Get Endpoints
func (c *Endpoints) Get(name string) (*api.Endpoints, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, constants.ResourceTypeEndpoints, name)
	endpointMsg := message.BuildMsg(modules.MetaGroup, "", constant.ModuleNameEdgeMesh, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(endpointMsg)
	if err != nil {
		return nil, fmt.Errorf("get endpointMsg from metaManager failed, err: %v", err)
	}

	var content []byte
	switch msg.Content.(type) {
	case []byte:
		content = msg.GetContent().([]byte)
	default:
		content, err = json.Marshal(msg.Content)
		if err != nil {
			return nil, fmt.Errorf("marshal message to endpointMsg failed, err: %v", err)
		}
	}

	if msg.GetOperation() == model.ResponseOperation {
		return handleEndpointFromMetaDB(content)
	}
	return handleEndpointFromMetaManager(content)
}

func handleEndpointFromMetaDB(content []byte) (*api.Endpoints, error) {
	var lists []string
	err := json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Endpoints list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("endpoints length from meta db is %d", len(lists))
	}

	var Endpoints api.Endpoints
	err = json.Unmarshal([]byte(lists[0]), &Endpoints)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Endpoint from db failed, err: %v", err)
	}
	return &Endpoints, nil
}

func handleEndpointFromMetaManager(content []byte) (*api.Endpoints, error) {
	var Endpoints api.Endpoints
	err := json.Unmarshal(content, &Endpoints)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Endpoint failed, err: %v", err)
	}
	return &Endpoints, nil
}
