package provider

import (
	"k8s.io/klog/v2"

	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
)

type TargetFactory interface {
	Type() string
	GetTarget(ep *v1.RuleEndpoint, targetResource map[string]string) Target
}

type Target interface {
	Name() string
	GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error)
}

var (
	// Modules map
	targets map[string]TargetFactory
)

func init() {
	targets = make(map[string]TargetFactory)
}

// RegisterSource register module
func RegisterTarget(t TargetFactory) {
	targets[t.Type()] = t
	klog.V(4).Info("target " + t.Type() + " registered")
}

// get source map
func GetTargetFactory(name string) (TargetFactory, bool) {
	target, exist := targets[name]
	return target, exist
}
