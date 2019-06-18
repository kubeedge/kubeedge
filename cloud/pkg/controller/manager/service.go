package manager

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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
func NewServiceManager(kubeClient *kubernetes.Clientset, namespace string) (*ServiceManager, error) {
	lw := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "services", namespace, fields.Everything())
	events := make(chan watch.Event, config.ServiceEventBuffer)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.Service{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &ServiceManager{events: events}, nil
}
