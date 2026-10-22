package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

// RuntimeClassManager manages all events of RuntimeClass by SharedInformer.
type RuntimeClassManager struct {
	events chan watch.Event
}

// Events returns the channel that saves events from watching RuntimeClass changes.
func (rm *RuntimeClassManager) Events() chan watch.Event {
	return rm.events
}

// NewRuntimeClassManager creates a RuntimeClassManager from a SharedIndexInformer.
// RuntimeClass is cluster-scoped, so no namespace filtering is required.
func NewRuntimeClassManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*RuntimeClassManager, error) {
	// Reuse ConfigMapEvent buffer size — RuntimeClass changes are similarly low-volume.
	events := make(chan watch.Event, config.Buffer.ConfigMapEvent)
	rh := NewCommonResourceEventHandler(events, nil)
	if _, err := si.AddEventHandler(rh); err != nil {
		return nil, err
	}

	return &RuntimeClassManager{events: events}, nil
}
