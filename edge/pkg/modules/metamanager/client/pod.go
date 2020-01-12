package client

import api "k8s.io/api/core/v1"

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
	send      SendInterface
}

func newPods(namespace string, s SendInterface) *pods {
	return &pods{
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
