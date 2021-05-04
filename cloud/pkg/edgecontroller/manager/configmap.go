package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
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
func NewConfigMapManager(si cache.SharedIndexInformer) (*ConfigMapManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.ConfigMapEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &ConfigMapManager{events: events}, nil
}
