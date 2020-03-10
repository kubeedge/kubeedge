package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const (
	defaultInternalMqttURL  = "tcp://127.0.0.1:1884"
	defaultExternalMqttURL  = "tcp://127.0.0.1:1883"
	defaultQos              = 0
	defaultRetain           = false
	defaultSessionQueueSize = 100
)

const (
	InternalMqttMode = iota // 0: launch an internal mqtt broker.
	BothMqttMode            // 1: launch an internal and external mqtt broker.
	ExternalMqttMode        // 2: launch an external mqtt broker.
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
