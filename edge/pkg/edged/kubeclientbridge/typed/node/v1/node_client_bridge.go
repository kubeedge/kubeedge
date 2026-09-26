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
	fakenodev1 "k8s.io/client-go/kubernetes/typed/node/v1/fake"
	nodev1 "k8s.io/client-go/kubernetes/typed/node/v1"
	"k8s.io/client-go/rest"

	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge/broadcaster"
)

type NodeV1Bridge struct {
	fakenodev1.FakeNodeV1
	MetaClient  metaclient.CoreInterface
	Broadcaster broadcaster.EventBroadcaster
}

func (c *NodeV1Bridge) RuntimeClasses() nodev1.RuntimeClassInterface {
	return newRuntimeClassesBridge(c.FakeNodeV1.RuntimeClasses(), c.MetaClient, c.Broadcaster)
}

func (c *NodeV1Bridge) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
