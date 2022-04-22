package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.UpgradeController
}

func InitConfigure(dc *v1alpha1.UpgradeController) {
	once.Do(func() {
		Config = Configure{
			UpgradeController: *dc,
		}
	})
}
