package manager

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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
func NewEndpointsManager(kubeClient *kubernetes.Clientset, namespace string) (*EndpointsManager, error) {
	lw := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "endpoints", namespace, fields.Everything())
	events := make(chan watch.Event, config.EndpointsEventBuffer)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.Endpoints{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &EndpointsManager{events: events}, nil
}
