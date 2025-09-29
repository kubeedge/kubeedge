package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

type StreamRuleEndpointManager struct {
	events chan watch.Event
}

func (srem *StreamRuleEndpointManager) Events() chan watch.Event {
	return srem.events
}

func NewStreamRuleEndpointManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*StreamRuleEndpointManager, error) {
	events := make(chan watch.Event, config.Buffer.StreamRuleEndpointsEvent)
	rh := NewCommonResourceEventHandler(events, nil)
	si.AddEventHandler(rh)

	return &StreamRuleEndpointManager{events: events}, nil
}
