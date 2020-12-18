package manager

import (
	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// RuleEndpointManager manage all events of rule by SharedInformer
type RuleEndpointManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch secret change
func (rem *RuleEndpointManager) Events() chan watch.Event {
	return rem.events
}

// NewRuleEndpointManager create RuleEndpointManager by kube clientset and namespace
func NewRuleEndpointManager(crdClient *rest.RESTClient, namespace string) (*RuleEndpointManager, error) {
	lw := cache.NewListWatchFromClient(crdClient, "rule-endpoints", namespace, fields.Everything())
	events := make(chan watch.Event, config.Config.Buffer.RuleEndpointsEvent)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.RuleEndpoint{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &RuleEndpointManager{events: events}, nil
}
