package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.ServiceBus
}

func InitConfigure(s *v1alpha1.ServiceBus) {
	once.Do(func() {
		Config = Configure{
			ServiceBus: *s,
		}
	})
}
