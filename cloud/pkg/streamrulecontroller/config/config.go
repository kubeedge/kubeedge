package config

import (
	"sync"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.StreamRuleController
}

func InitConfigure(src *v1alpha1.StreamRuleController) {
	once.Do(func() {
		Config = Configure{
			StreamRuleController: *src,
		}
	})
}
