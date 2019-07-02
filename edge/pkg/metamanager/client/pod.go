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

// Package client
package client

import (
	"github.com/kubeedge/beehive/pkg/core/context"
	api "k8s.io/api/core/v1"
)

//PodsGetter is interface to get pods
type PodsGetter interface {
	Pods(namespace string) PodsInterface
}

//PodsInterface is pod interface
type PodsInterface interface {
	Create(*api.Pod) (*api.Pod, error)
	Update(*api.Pod) error
	Delete(name string) error
	Get(name string) (*api.Pod, error)
}

type pods struct {
	namespace string
	context   *context.Context
	send      SendInterface
}

func newPods(namespace string, c *context.Context, s SendInterface) *pods {
	return &pods{
		context:   c,
		send:      s,
		namespace: namespace,
	}
}

func (c *pods) Create(cm *api.Pod) (*api.Pod, error) {
	return nil, nil
}

func (c *pods) Update(cm *api.Pod) error {
	return nil
}

func (c *pods) Delete(name string) error {
	return nil
}

func (c *pods) Get(name string) (*api.Pod, error) {
	return nil, nil
}
