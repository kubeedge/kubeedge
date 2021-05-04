package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// ServiceManager manage all events of service by SharedInformer
type ServiceManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch service change
func (sm *ServiceManager) Events() chan watch.Event {
	return sm.events
}

// NewServiceManager create ServiceManager by kube clientset and namespace
func NewServiceManager(si cache.SharedIndexInformer) (*ServiceManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.ServiceEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &ServiceManager{events: events}, nil
}
