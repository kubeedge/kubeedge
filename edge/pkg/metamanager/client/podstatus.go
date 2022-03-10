package client

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
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
	// if edge and cloud connection is interrupted, we just return error to
	// prevent a large number of messages accumulating in the channel, and
	// mass message hits cloudCore when network connection is established
	// between edge and cloud. status manager will retry update pod status
	// until send successfully.
	if !connect.IsConnected() {
		return fmt.Errorf("edge and cloud connection is interrupted")
	}
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
