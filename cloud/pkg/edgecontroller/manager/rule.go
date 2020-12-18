package manager

import (
	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
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

// NewRuleManager create RuleManager by kube clientset and namespace
func NewRuleManager(crdClient *rest.RESTClient, namespace string) (*RuleManager, error) {
	lw := cache.NewListWatchFromClient(crdClient, "rules", namespace, fields.Everything())
	events := make(chan watch.Event, config.Config.Buffer.RulesEvent)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.Rule{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &RuleManager{events: events}, nil
}