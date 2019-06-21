package client

import (
	"github.com/kubeedge/beehive/pkg/core/context"
	api "k8s.io/api/core/v1"
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
	context   *context.Context
	send      SendInterface
}

func newNodes(namespace string, c *context.Context, s SendInterface) *nodes {
	return &nodes{
		context:   c,
		send:      s,
		namespace: namespace,
	}
}

func (c *nodes) Create(cm *api.Node) (*api.Node, error) {
	return nil, nil
}

func (c *nodes) Update(cm *api.Node) error {
	return nil
}

func (c *nodes) Delete(name string) error {
	return nil
}

func (c *nodes) Get(name string) (*api.Node, error) {
	return nil, nil
}
