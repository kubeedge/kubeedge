package client

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

//PodsGetter is interface to get pods
type PodsGetter interface {
	Pods(namespace string) PodsInterface
}

//PodsInterface is pod interface
type PodsInterface interface {
	Create(*corev1.Pod) (*corev1.Pod, error)
	Update(*corev1.Pod) error
	Delete(name string) error
	Get(name string) (*corev1.Pod, error)
}

type pods struct {
	namespace string
	send      SendInterface
}

func newPods(namespace string, s SendInterface) *pods {
	return &pods{
		send:      s,
		namespace: namespace,
	}
}

func (c *pods) Create(cm *corev1.Pod) (*corev1.Pod, error) {
	return nil, nil
}

func (c *pods) Update(cm *corev1.Pod) error {
	return nil
}

func (c *pods) Delete(name string) error {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePod, name)
	podDeleteMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.DeleteOperation, nil)
	c.send.Send(podDeleteMsg)
	return nil
}

func (c *pods) Get(name string) (*corev1.Pod, error) {
	return nil, nil
}
