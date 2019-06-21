package client

import (
	"fmt"
	"github.com/kubeedge/beehive/pkg/core/context"
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
	context   *context.Context
	send      SendInterface
}

func newPodStatus(namespace string, c *context.Context, s SendInterface) *podStatus {
	return &podStatus{
		context:   c,
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
