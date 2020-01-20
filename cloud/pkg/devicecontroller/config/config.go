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

func InitConfigure(dc *v1alpha1.DeviceController, kubeAPIConfig *v1alpha1.KubeAPIConfig) {
	once.Do(func() {
		c = Configure{
			DeviceController: *dc,
			KubeAPIConfig:    *kubeAPIConfig,
		}
	})
}

func Get() *Configure {
	return &c
}
