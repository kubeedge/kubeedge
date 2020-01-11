package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

var c Configure
var once sync.Once

type Configure struct {
	KubeAPIConfig *v1alpha1.KubeAPIConfig
}

func InitConfigure(kubeAPIConfig *v1alpha1.KubeAPIConfig) {
	once.Do(func() {
		c = Configure{
			KubeAPIConfig: kubeAPIConfig,
		}
	})
}

func Get() *Configure {
	return &c
}
