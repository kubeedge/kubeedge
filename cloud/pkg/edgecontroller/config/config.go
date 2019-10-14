package config

import (
	"sync"

	"github.com/kubeedge/beehive/pkg/core/context"
	cconfig "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
	econfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
	sideconfig "github.com/kubeedge/kubeedge/pkg/edgesite/apis/config"
)

var (
	Context *context.Context
	c       Config
	once    sync.Once
)

func InitEdgeControllerConfig(econtroller *cconfig.EdgeControllerConfig,
	kube *cconfig.KubeConfig,
	cc *cconfig.ControllerContext,
	ec *econfig.EdgedConfig,
	m *sideconfig.Metamanager) {
	once.Do(func() {
		c.EdgeController = *econtroller
		c.Kube = *kube
		c.ContextController = *cc
		c.EdgedConfig = *ec
		c.EdgeSiteEnabled = m.EdgeSite
	})
}

type Config struct {
	EdgeController    cconfig.EdgeControllerConfig
	Kube              cconfig.KubeConfig
	ContextController cconfig.ControllerContext
	EdgedConfig       econfig.EdgedConfig
	EdgeSiteEnabled   bool
}

func Conf() *Config {
	return &c
}
