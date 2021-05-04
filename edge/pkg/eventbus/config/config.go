package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EventBus
	NodeName string
}

func InitConfigure(eventbus *v1alpha1.EventBus, nodeName string) {
	once.Do(func() {
		Config = Configure{
			EventBus: *eventbus,
			NodeName: nodeName,
		}
	})
}
