package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// SecretManager manage all events of secret by SharedInformer
type SecretManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch secret change
func (sm *SecretManager) Events() chan watch.Event {
	return sm.events
}

// NewSecretManager create SecretManager by kube clientset and namespace
func NewSecretManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*SecretManager, error) {
	events := make(chan watch.Event, config.Buffer.SecretEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &SecretManager{events: events}, nil
}
