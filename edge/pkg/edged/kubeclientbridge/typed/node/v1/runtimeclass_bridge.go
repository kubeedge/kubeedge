/*
Copyright 2024 The KubeEdge Authors.

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

package v1

import (
	"context"

	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"
	typednodev1 "k8s.io/client-go/kubernetes/typed/node/v1"

	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge/broadcaster"
)

var _ typednodev1.RuntimeClassInterface = &RuntimeClassesBridge{}

type RuntimeClassesBridge struct {
	typednodev1.RuntimeClassInterface
	metaClient  metaclient.CoreInterface
	broadcaster broadcaster.EventBroadcaster
}

func newRuntimeClassesBridge(
	fakeClient typednodev1.RuntimeClassInterface,
	metaClient metaclient.CoreInterface,
	broadcaster broadcaster.EventBroadcaster,
) *RuntimeClassesBridge {
	return &RuntimeClassesBridge{
		RuntimeClassInterface: fakeClient,
		metaClient:            metaClient,
		broadcaster:           broadcaster,
	}
}

func (c *RuntimeClassesBridge) Get(
	ctx context.Context,
	name string,
	opts metav1.GetOptions,
) (*nodev1.RuntimeClass, error) {
	return c.metaClient.RuntimeClasses().Get(name)
}

func (c *RuntimeClassesBridge) List(
	ctx context.Context,
	opts metav1.ListOptions,
) (*nodev1.RuntimeClassList, error) {
	rcs, err := c.metaClient.RuntimeClasses().List()
	if err != nil {
		return nil, err
	}

	return &nodev1.RuntimeClassList{
		Items: rcs,
	}, nil
}

func (c *RuntimeClassesBridge) Watch(
	ctx context.Context,
	opts metav1.ListOptions,
) (watch.Interface, error) {
	if c.broadcaster == nil {
		return c.RuntimeClassInterface.Watch(ctx, opts)
	}

	return c.broadcaster.Subscribe(), nil
}
