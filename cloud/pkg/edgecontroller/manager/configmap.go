package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
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
func NewConfigMapManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*ConfigMapManager, error) {
	events := make(chan watch.Event, config.Buffer.ConfigMapEvent)
	rh := NewCommonResourceEventHandler(events, nil)
	if _, err := si.AddEventHandler(rh); err != nil {
		return nil, err
	}

	return &ConfigMapManager{events: events}, nil
}
