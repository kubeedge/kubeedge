package config

import (
	"sync"

	"github.com/kubeedge/beehive/pkg/core/context"
	cloudconfig "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
)

var (
	c    Config
	once sync.Once
	// Context is beehive context used to send message
	Context *context.Context
)

func InitDeviceControllerConfig(cc *cloudconfig.ControllerContext, k *cloudconfig.KubeConfig) {
	once.Do(func() {
		c.ControllerContext = *cc
		c.Kube = *k
	})
}

type Config struct {
	ControllerContext cloudconfig.ControllerContext
	Kube              cloudconfig.KubeConfig
}

func Conf() *Config {
	return &c
}
