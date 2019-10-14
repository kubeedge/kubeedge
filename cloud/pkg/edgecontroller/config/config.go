package config

import (
	"sync"

	"github.com/kubeedge/beehive/pkg/core/context"
	cconfig "github.com/kubeedge/kubeedge/cloud/pkg/apis/cloudcore/config"
)

// Context ...
var Context *context.Context

var config Config
var once sync.Once

func InitEdgeControllerConfig(c *cconfig.CloudCoreConfig) {
	once.Do(func() {
		config.EdgeController = *(c.EdgeController)
		config.Kube = *(c.Kube)
	})
}

type Config struct {
	EdgeController cconfig.EdgeControllerConfig
	Kube           cconfig.KubeConfig
}

func Conf() *Config {
	return &config
}
