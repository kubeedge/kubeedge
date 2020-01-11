package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

var c Configure
var once sync.Once

type Configure struct {
	v1alpha1.DeviceController
	KubeAPIConfig v1alpha1.KubeAPIConfig
}

func InitConfigure(controller *v1alpha1.DeviceController, k *v1alpha1.KubeAPIConfig) {
	once.Do(func() {
		c = Configure{
			DeviceController: *controller,
			KubeAPIConfig:    *k,
		}
	})
}

func Get() *Configure {
	return &c
}
