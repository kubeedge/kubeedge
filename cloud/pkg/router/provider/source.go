package provider

import (
	"k8s.io/klog/v2"

	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
)

type SourceFactory interface {
	Type() string
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
	sources map[string]SourceFactory
)

func init() {
	sources = make(map[string]SourceFactory)
}

// RegisterSource register module
func RegisterSource(s SourceFactory) {
	sources[s.Type()] = s
	klog.V(4).Info("source " + s.Type() + " registered")
}

// get source map
func GetSourceFactory(name string) (SourceFactory, bool) {
	source, exist := sources[name]
	return source, exist
}
