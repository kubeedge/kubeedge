package config

import (
	"sync"

	"github.com/kubeedge/beehive/pkg/core/context"
	cconfig "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
	econfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

var (
	Context *context.Context
	c       Config
	once    sync.Once
)

func InitEdgeControllerConfig(cc *cconfig.CloudCoreConfig, ec *econfig.EdgedConfig) {
	once.Do(func() {
		c.EdgeController = *(cc.EdgeController)
		c.Kube = *(cc.Kube)
		c.ContextController = *(cc.ControllerContext)
		c.EdgedConfig = *ec
	})
}

type Config struct {
	EdgeController    cconfig.EdgeControllerConfig
	Kube              cconfig.KubeConfig
	ContextController cconfig.ControllerContext
	EdgedConfig       econfig.EdgedConfig
}

func Conf() *Config {
	return &c
}
