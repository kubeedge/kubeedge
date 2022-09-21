package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"sync"
)

// RelayRCManager is a manager watch relayrc change event
type RelayRCManager struct {
	// events from watch kubernetes api server
	events    chan watch.Event
	RelayInfo sync.Map
}

// Events return a channel, can receive all device event
func (rrm *RelayRCManager) Events() chan watch.Event {
	return rrm.events
}

// NewRelayRCManager create RelayRCManagerManager
func NewRelayRCManager(si cache.SharedIndexInformer) (*RelayRCManager, error) {
	events := make(chan watch.Event)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &RelayRCManager{events: events}, nil
}
