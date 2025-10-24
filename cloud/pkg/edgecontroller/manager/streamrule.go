package manager

import (
	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type StreamRuleManager struct {
	events chan watch.Event
}

func (srm *StreamRuleManager) Events() chan watch.Event {
	return srm.events
}

func NewStreamRuleManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*StreamRuleManager, error) {
	events := make(chan watch.Event, config.Buffer.StreamRulesEvent)
	rh := NewCommonResourceEventHandler(events, nil)
	si.AddEventHandler(rh)

	return &StreamRuleManager{events: events}, nil
}
