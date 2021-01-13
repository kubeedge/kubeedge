package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.DeviceController
}

func InitConfigure(dc *v1alpha1.DeviceController) {
	once.Do(func() {
		Config = Configure{
			DeviceController: *dc,
		}
	})
}
