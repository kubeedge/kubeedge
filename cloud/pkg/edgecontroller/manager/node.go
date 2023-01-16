package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// NodesManager manage all events of nodes by SharedInformer
type NodesManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch nodes change
func (nm *NodesManager) Events() chan watch.Event {
	return nm.events
}

// NewNodesManager create NodesManager by kube clientset and namespace
func NewNodesManager(si cache.SharedIndexInformer) (*NodesManager, error) {
	events := make(chan watch.Event)
	rh := NewCommonResourceEventHandler(events, nil)
	si.AddEventHandler(rh)

	return &NodesManager{events: events}, nil
}
