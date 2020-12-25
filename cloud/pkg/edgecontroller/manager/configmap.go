package manager

import (
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
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
func NewConfigMapManager() (*ConfigMapManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.ConfigMapEvent)
	rh := NewCommonResourceEventHandler(events)
	si := informers.GetGlobalInformers().ConfigMap()
	si.AddEventHandler(rh)

	return &ConfigMapManager{events: events}, nil
}
