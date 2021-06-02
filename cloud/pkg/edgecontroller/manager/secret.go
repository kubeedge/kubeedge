package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
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
func NewSecretManager(si cache.SharedIndexInformer) (*SecretManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.SecretEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &SecretManager{events: events}, nil
}
