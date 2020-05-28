package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

// Connected stands for whether it is connected
// TODO need consider to add lock @kadisi
var Connected = false

type Configure struct {
	v1alpha1.MetaManager
}

func InitConfigure(m *v1alpha1.MetaManager) {
	once.Do(func() {
		Config = Configure{
			MetaManager: *m,
		}
	})
}
