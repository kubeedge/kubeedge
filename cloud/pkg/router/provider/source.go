package provider

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	v1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
)

type SourceFactory interface {
	Type() v1.RuleEndpointTypeDef
	GetSource(ep *v1.RuleEndpoint, sourceResource map[string]string) Source
}

type Source interface {
	Name() string
	RegisterListener(handle listener.Handle) error
	UnregisterListener()
	Forward(Target, interface{}) (interface{}, error)
}

var (
	// Modules map
	sources map[v1.RuleEndpointTypeDef]SourceFactory
)

func init() {
	sources = make(map[v1.RuleEndpointTypeDef]SourceFactory)
}

// RegisterSource register module
func RegisterSource(s SourceFactory) {
	sources[s.Type()] = s
	klog.V(4).Info("source " + s.Type() + " registered")
}

// get source map
func GetSourceFactory(name v1.RuleEndpointTypeDef) (SourceFactory, bool) {
	source, exist := sources[name]
	return source, exist
}
