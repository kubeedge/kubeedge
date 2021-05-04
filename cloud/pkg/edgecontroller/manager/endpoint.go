package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// EndpointsManager manage all events of endpoints by SharedInformer
type EndpointsManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch endpoints change
func (sm *EndpointsManager) Events() chan watch.Event {
	return sm.events
}

// NewEndpointsManager create EndpointsManager by kube clientset and namespace
func NewEndpointsManager(si cache.SharedIndexInformer) (*EndpointsManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.EndpointsEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &EndpointsManager{events: events}, nil
}
