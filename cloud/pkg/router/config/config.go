package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.Router
}

func InitConfigure(router *v1alpha1.Router) {
	once.Do(func() {
		Config = Configure{
			Router: *router,
		}
	})
}
