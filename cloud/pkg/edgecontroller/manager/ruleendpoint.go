package manager

import (
	"fmt"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
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
func NewRuleEndpointManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*RuleEndpointManager, error) {
	if config == nil {
		klog.Errorf("nil config error")
		return nil, fmt.Errorf("nil config error")
	}
	events := make(chan watch.Event, config.Buffer.RuleEndpointsEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &RuleEndpointManager{events: events}, nil
}
