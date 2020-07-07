package manager

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	//"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// PersistentvolumeclaimManager manage all events of Persistentvolumeclaim by SharedInformer
type PersistentvolumeclaimManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch Persistentvolumeclaim change
func (sm *PersistentvolumeclaimManager) Events() chan watch.Event {
	return sm.events
}

// NewPersistentvolumeclaimManager create PersistentvolumeclaimManager by kube clientset and namespace
func NewPersistentvolumeclaimManager(kubeClient *kubernetes.Clientset, namespace string) (*PersistentvolumeclaimManager, error) {
	lw := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "persistentvolumeclaims", namespace, fields.Everything())
	events := make(chan watch.Event, 1)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.PersistentVolumeClaim{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &PersistentvolumeclaimManager{events: events}, nil
}
