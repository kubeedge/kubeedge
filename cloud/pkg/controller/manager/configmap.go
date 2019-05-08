package manager

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ConfigMapManager manage all events of configmap by SharedInformer
type ConfigMapManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch configmap change
func (cmm *ConfigMapManager) Events() chan watch.Event {
	return cmm.events
}

// NewConfigMapManager create ConfigMapManager by kube clientset and namespace
func NewConfigMapManager(kubeClient *kubernetes.Clientset, namespace string) (*ConfigMapManager, error) {
	lw := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "configmaps", namespace, fields.Everything())
	events := make(chan watch.Event, config.ConfigMapEventBuffer)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.ConfigMap{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &ConfigMapManager{events: events}, nil
}
