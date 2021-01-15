package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// RuleEndpointManager manage all events of rule by SharedInformer
type RuleEndpointManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch secret change
func (rem *RuleEndpointManager) Events() chan watch.Event {
	return rem.events
}

// NewRuleEndpointManager create RuleEndpointManager by SharedIndexInformer
func NewRuleEndpointManager(si cache.SharedIndexInformer) (*RuleEndpointManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.RuleEndpointsEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &RuleEndpointManager{events: events}, nil
}
