package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
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
func NewEndpointsManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*EndpointsManager, error) {
	events := make(chan watch.Event, config.Buffer.EndpointsEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &EndpointsManager{events: events}, nil
}
