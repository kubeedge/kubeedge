package manager

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// PersistentvolumeManager manage all events of Persistentvolumeby SharedInformer
type PersistentvolumeManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch Persistentvolume change
func (sm *PersistentvolumeManager) Events() chan watch.Event {
	return sm.events
}

// NewPersistentvolumeManager create PersistentvolumeManager by kube clientset and namespace
func NewPersistentvolumeManager(kubeClient *kubernetes.Clientset, namespace string) (*PersistentvolumeManager, error) {
	lw := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "persistentvolumes", namespace, fields.Everything())
	events := make(chan watch.Event, 1)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.PersistentVolume{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &PersistentvolumeManager{events: events}, nil
}
