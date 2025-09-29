package provider

import (
	"github.com/kubeedge/api/apis/streamrules/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/listener"
	"k8s.io/klog/v2"
)

type TargetFactory interface {
	Type() v1alpha1.ProtocolType
	GetTarget(ep *v1alpha1.StreamRuleEndpoint, targetResource map[string]string) Target
}

type Target interface {
	Name() string
	RegisterListener(handle listener.Handle) error
	UnregisterListener()
	SendMsg(interface{}) (interface{}, error)
}

type Targets []Target

var (
	// Modules map
	targets map[v1alpha1.ProtocolType]TargetFactory
)

func init() {
	targets = make(map[v1alpha1.ProtocolType]TargetFactory)
}

// RegisterTarget register module
func RegisterTarget(t TargetFactory) {
	targets[t.Type()] = t
	klog.Info("target " + string(t.Type()) + " registered")
}

// get targets map
func GetTargetFactory(name v1alpha1.ProtocolType) (TargetFactory, bool) {
	target, exist := targets[name]
	return target, exist
}
