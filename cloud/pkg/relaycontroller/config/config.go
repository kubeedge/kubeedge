package config

import (
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"sync"
)

var Config Configure
var once sync.Once

type Configure struct {
	RelayController *configv1alpha1.RelayController
}

func InitConfigure(rc *configv1alpha1.RelayController) {
	once.Do(func() {
		Config = Configure{
			RelayController: rc,
		}
	})
}
