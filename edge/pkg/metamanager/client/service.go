package client

import (
	v1 "k8s.io/api/core/v1"
)

// ServiceGetter interface
type ServiceGetter interface {
	Services(namespace string) ServiceInterface
}

// ServiceInterface is an interface
type ServiceInterface interface {
	Create(*v1.Service) (*v1.Service, error)
	Update(service *v1.Service) error
	Delete(name string) error
	Get(name string) (*v1.Service, error)
	GetPods(name string) ([]v1.Pod, error)
	ListAll() ([]v1.Service, error)
}

type services struct {
	namespace string
	send      SendInterface
}

func newServices(namespace string, s SendInterface) *services {
	return &services{
		namespace: namespace,
		send:      s,
	}
}

func (s *services) Create(*v1.Service) (*v1.Service, error) {
	return &v1.Service{}, nil
}

func (s *services) Update(service *v1.Service) error {
	return nil
}

func (s *services) Delete(name string) error {
	return nil
}

func (s *services) GetPods(name string) {
}

func (s *services) Get(name string) {
}

func (s *services) ListAll() {
}
