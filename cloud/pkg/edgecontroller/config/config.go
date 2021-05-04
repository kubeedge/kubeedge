package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeController
}

func InitConfigure(ec *v1alpha1.EdgeController) {
	once.Do(func() {
		Config = Configure{
			EdgeController: *ec,
		}
	})
}
