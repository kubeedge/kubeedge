package manager

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// RuleManager manage all events of rule by SharedInformer
type RuleManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch secret change
func (rm *RuleManager) Events() chan watch.Event {
	return rm.events
}

// NewRuleManager create RuleManager by SharedIndexInformer
func NewRuleManager(si cache.SharedIndexInformer) (*RuleManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.RulesEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &RuleManager{events: events}, nil
}