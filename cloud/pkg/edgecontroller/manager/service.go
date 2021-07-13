package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
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
func NewServiceManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*ServiceManager, error) {
	events := make(chan watch.Event, config.Buffer.ServiceEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &ServiceManager{events: events}, nil
}
