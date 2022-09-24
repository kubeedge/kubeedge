package config

import (
	"sync"

	edgehubconfig "github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha2.MetaServer
	NodeName string
}

func InitConfigure(c *v1alpha2.MetaServer) {
	once.Do(func() {
		Config = Configure{
			MetaServer: *c,
			// so edgehub must register before metamanager
			NodeName: edgehubconfig.Config.NodeName,
		}
	})
}
