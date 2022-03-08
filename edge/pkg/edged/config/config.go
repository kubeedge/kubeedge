package config

import (
	"sync"

	"k8s.io/component-base/featuregate"
	"k8s.io/kubernetes/pkg/kubelet"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.Edged
	*kubelet.Dependencies
	featuregate.FeatureGate
}

func InitConfigure(e *v1alpha1.Edged) {
	once.Do(func() {
		Config = Configure{
			Edged: *e,
		}
	})
}
