package config

import (
	"sync"

	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha2.MetaManager
}

func InitConfigure(m *v1alpha2.MetaManager) {
	once.Do(func() {
		Config = Configure{
			MetaManager: *m,
		}
		metaserverconfig.InitConfigure(Config.MetaServer)
	})
}
