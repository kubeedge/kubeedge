package manager

import (
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
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
func NewSecretManager() (*SecretManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.SecretEvent)
	rh := NewCommonResourceEventHandler(events)
	si := informers.GetGlobalInformers().Secrets()
	si.AddEventHandler(rh)

	return &SecretManager{events: events}, nil
}
