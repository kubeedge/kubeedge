package client

import (
	api "k8s.io/api/core/v1"
)

// EndpointsGetter has a method to return a EndpointsInterface.
// A group's client should implement this interface.
type EndpointsGetter interface {
	Endpoints(namespace string) EndpointsInterface
}

// EndpointsInterface has methods to work with Endpoints resources.
type EndpointsInterface interface {
	Create(*api.Endpoints) (*api.Endpoints, error)
	Update(*api.Endpoints) error
	Delete(name string) error
	Get(name string) (*api.Endpoints, error)
}

// Endpoints is struct implementing EndpointsInterface
type Endpoints struct {
	namespace string
	send      SendInterface
}

func newEndpoints(namespace string, s SendInterface) *Endpoints {
	return &Endpoints{
		send:      s,
		namespace: namespace,
	}
}

// Create Endpoints
func (c *Endpoints) Create(cm *api.Endpoints) (*api.Endpoints, error) {
	return nil, nil
}

// Update Endpoints
func (c *Endpoints) Update(cm *api.Endpoints) error {
	return nil
}

// Delete Endpoints
func (c *Endpoints) Delete(name string) error {
	return nil
}

// Get Endpoints
func (c *Endpoints) Get(name string) {
}
