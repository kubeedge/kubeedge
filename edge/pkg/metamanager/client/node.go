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
