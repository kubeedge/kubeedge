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
	commodule "github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

//PodStatusGetter is interface to get pod status
type PodStatusGetter interface {
	PodStatus(namespace string) PodStatusInterface
}

//PodStatusInterface is interface of pod status
type PodStatusInterface interface {
	Create(*edgeapi.PodStatusRequest) (*edgeapi.PodStatusRequest, error)
	Update(rsName string, ps edgeapi.PodStatusRequest) error
	Delete(name string) error
	Get(name string) (*edgeapi.PodStatusRequest, error)
}

type podStatus struct {
	namespace string
	send      SendInterface
}

func newPodStatus(namespace string, s SendInterface) *podStatus {
	return &podStatus{
		send:      s,
		namespace: namespace,
	}
}

func (c *podStatus) Create(ps *edgeapi.PodStatusRequest) (*edgeapi.PodStatusRequest, error) {
	return nil, nil
}

func (c *podStatus) Update(rsName string, ps edgeapi.PodStatusRequest) error {
	podStatusMsg := message.BuildMsg(commodule.MetaGroup, "", commodule.EdgedModuleName, c.namespace+"/"+model.ResourceTypePodStatus+"/"+rsName, model.UpdateOperation, ps)
	_, err := c.send.SendSync(podStatusMsg)
	if err != nil {
		return fmt.Errorf("update podstatus failed, err: %v", err)
	}

	return nil
}

func (c *podStatus) Delete(name string) error {
	return nil
}

func (c *podStatus) Get(name string) (*edgeapi.PodStatusRequest, error) {
	return nil, nil
}
