package manager

import (
	"fmt"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
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
func NewRuleManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*RuleManager, error) {
	if config == nil {
		klog.Errorf("nil config error")
		return nil, fmt.Errorf("nil config error")
	}
	events := make(chan watch.Event, config.Buffer.RulesEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &RuleManager{events: events}, nil
}
